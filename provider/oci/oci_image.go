//go:build oci || !onlyprovider

package oci

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/objectstorage"
	"github.com/oracle/oci-go-sdk/workrequests"
	"github.com/schollz/progressbar/v3"
)

var (
	qcow2ImagesDir = lepton.GetOpsHome() + "/qcow2-images"
)

// BuildImage creates local image
func (p *Provider) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	imagePath, err := p.createQcow2Image(c)
	if err != nil {
		return "", err
	}

	err = os.Remove(c.RunConfig.ImageName)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

func (p *Provider) createQcow2Image(c *types.Config) (imagePath string, err error) {
	qcow2ImagesDir, err := findOrCreateqcow2ImagesDir()
	if err != nil {
		return
	}

	imagePath = path.Join(qcow2ImagesDir, c.CloudConfig.ImageName+".qcow2")

	args := []string{
		"convert",
		"-O", "qcow2",
		c.RunConfig.ImageName, imagePath,
	}

	cmd := exec.Command("qemu-img", args...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return
	}

	return
}

func findOrCreateqcow2ImagesDir() (string, error) {
	if _, err := os.Stat(qcow2ImagesDir); os.IsNotExist(err) {
		os.MkdirAll(qcow2ImagesDir, 0755)
	} else if err != nil {
		return "", err
	}

	return qcow2ImagesDir, nil
}

// BuildImageWithPackage creates local image using package image
func (p *Provider) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}

	imagePath, err := p.createQcow2Image(c)
	if err != nil {
		return "", err
	}

	err = os.Remove(c.RunConfig.ImageName)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// CreateImage creates a storage object and upload image
func (p *Provider) CreateImage(ctx *lepton.Context, imagePath string) (err error) {
	bucketNamespace := ctx.Config().CloudConfig.BucketNamespace
	bucketName := ctx.Config().CloudConfig.BucketName
	imageName := ctx.Config().CloudConfig.ImageName

	if bucketName == "" {
		return errors.New("specify the bucket name in cloud configuration")
	}

	if bucketNamespace == "" {
		return errors.New("specify the bucket namespace in cloud configuration. Access bucket details page to get the namespace")
	}

	image, err := p.fileSystem.Open(imagePath)
	if err != nil {
		ctx.Logger().Error(err)
		return fmt.Errorf("failed reading file %s", imagePath)
	}

	imageStats, err := image.Stat()
	if err != nil {
		ctx.Logger().Error(err)
		return fmt.Errorf("failed getting file stats of %s", imagePath)
	}

	imageSize := imageStats.Size()

	_, err = p.storageClient.PutObject(context.TODO(), objectstorage.PutObjectRequest{
		NamespaceName: &bucketNamespace,
		BucketName:    &bucketName,
		ContentLength: &imageSize,
		ObjectName:    &imageName,
		PutObjectBody: image,
	})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed uploading image")
	}

	job, err := p.computeClient.CreateImage(context.TODO(), core.CreateImageRequest{
		CreateImageDetails: core.CreateImageDetails{
			CompartmentId: &p.compartmentID,
			DisplayName:   &imageName,
			FreeformTags:  ociOpsTags,
			LaunchMode:    core.CreateImageDetailsLaunchModeParavirtualized,
			ImageSourceDetails: core.ImageSourceViaObjectStorageTupleDetails{
				NamespaceName:   &bucketNamespace,
				BucketName:      &bucketName,
				ObjectName:      &imageName,
				SourceImageType: core.ImageSourceDetailsSourceImageTypeQcow2,
			},
		},
	})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed importing image from storage")
	}

	fmt.Println("It will take a while to import the image. Feel free to exit (Control+C), it will not stop the image importing.")
	bar := progressbar.New(100)
	bar.RenderBlank()

	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan bool)
	getProgress := func() {
		var res workrequests.GetWorkRequestResponse
		res, err = p.workRequestClient.GetWorkRequest(context.TODO(), workrequests.GetWorkRequestRequest{WorkRequestId: job.OpcWorkRequestId})
		if err != nil {
			quit <- true
			return
		}

		if *res.PercentComplete == 100 {
			quit <- true
		}

		bar.Set(int(*res.PercentComplete))
		bar.RenderBlank()
	}
	for {
		go getProgress()
		select {
		case <-ticker.C:
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

// ListImages prints oci images in table format
func (p *Provider) ListImages(ctx *lepton.Context, filter string) error {
	images, err := p.GetImages(ctx, filter)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Size", "CreatedAt"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, i := range images {
		var row []string
		row = append(row, i.Name)
		row = append(row, i.Status)
		row = append(row, lepton.Bytes2Human(i.Size))
		row = append(row, lepton.Time2Human(i.Created))
		table.Append(row)
	}
	table.Render()
	return nil
}

// GetImages returns the list of images
func (p *Provider) GetImages(ctx *lepton.Context, filter string) (images []lepton.CloudImage, err error) {
	images = []lepton.CloudImage{}

	imagesList, err := p.computeClient.ListImages(context.TODO(), core.ListImagesRequest{OperatingSystem: types.StringPtr("Custom"), CompartmentId: types.StringPtr(p.compartmentID)})
	if err != nil {
		ctx.Logger().Error(err)
		return nil, errors.New("failed getting images")
	}

	for _, i := range imagesList.Items {
		if checkHasOpsTags(i.FreeformTags) {
			cimage := lepton.CloudImage{
				ID:      *i.Id,
				Name:    *i.DisplayName,
				Status:  string(i.LifecycleState),
				Created: i.TimeCreated.Time,
			}
			if i.SizeInMBs != nil {
				cimage.Size = *i.SizeInMBs / 1024
			}

			images = append(images, cimage)
		}
	}

	return
}

// DeleteImage removes oci image
func (p *Provider) DeleteImage(ctx *lepton.Context, imagename string) (err error) {

	image, err := p.getImageByName(ctx, imagename)
	if err != nil {
		return
	}

	_, err = p.computeClient.DeleteImage(context.TODO(), core.DeleteImageRequest{ImageId: &image.ID})

	return
}

func (p *Provider) getImageByName(ctx *lepton.Context, name string) (*lepton.CloudImage, error) {
	images, err := p.GetImages(ctx, "")
	if err != nil {
		return nil, err
	}

	for _, i := range images {
		if i.Name == name {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("image with name %s not found", name)
}

// ResizeImage is a stub
func (p *Provider) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return errors.New("Unsupported")
}

// SyncImage is a stub
func (p *Provider) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return errors.New("Unsupported")
}

// CustomizeImage is a stub
func (p *Provider) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", errors.New("Unsupported")
}
