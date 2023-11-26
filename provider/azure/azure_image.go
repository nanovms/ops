//go:build azure || !onlyprovider

package azure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-07-02/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on Azure
func (a *Azure) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	a.hyperVGen = compute.HyperVGenerationTypesV1
	imageType := strings.ToLower(c.CloudConfig.ImageType)
	if imageType != "" {
		if imageType == "gen1" {
		} else if imageType == "gen2" {
			c.Uefi = true
			a.hyperVGen = compute.HyperVGenerationTypesV2
		} else {
			return "", fmt.Errorf("invalid image type '%s'; available types: 'gen1', 'gen2'", c.CloudConfig.ImageType)
		}
	}
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return a.CustomizeImage(ctx)
}

func (a *Azure) getArchiveName(ctx *lepton.Context) string {
	return ctx.Config().CloudConfig.ImageName + ".tar.gz"
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (a *Azure) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")

	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return "", fmt.Errorf("failed to unlink: %+v", err)
		}
	}

	err := os.Link(imagePath, symlink)
	if err != nil {
		return "", err
	}

	archPath := filepath.Join(filepath.Dir(imagePath), a.getArchiveName(ctx))
	files := []string{symlink}

	err = lepton.CreateArchive(archPath, files)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// BuildImageWithPackage to upload on Azure
func (a *Azure) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {

	c := ctx.Config()

	a.hyperVGen = compute.HyperVGenerationTypesV1
	imageType := strings.ToLower(c.CloudConfig.ImageType)
	if imageType != "" {
		if imageType == "gen1" {
		} else if imageType == "gen2" {
			c.Uefi = true
			a.hyperVGen = compute.HyperVGenerationTypesV2
		} else {
			return "", fmt.Errorf("invalid image type '%s'; available types: 'gen1', 'gen2'", c.CloudConfig.ImageType)
		}
	}

	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return a.CustomizeImage(ctx)
}

// CreateImage - Creates image on Azure using nanos images
func (a *Azure) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := a.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	imagesClient := a.getImagesClient()

	c := ctx.Config()
	imgName := c.CloudConfig.ImageName

	bucket, err := a.getBucketName()
	if err != nil {
		return err
	}

	region := a.getLocation(ctx.Config())
	container := "quickstart-nanos"
	disk := c.CloudConfig.ImageName + ".vhd"

	uri := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	imageParams := compute.Image{
		Location: to.StringPtr(region),
		Tags:     getAzureDefaultTags(),
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					OsType:  compute.OperatingSystemTypesLinux,
					BlobURI: to.StringPtr(uri),
					OsState: compute.Generalized,
				},
			},
			HyperVGeneration: a.hyperVGen,
		},
	}

	images, err := a.GetImages(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < len(images); i++ {
		if images[i].Name == imgName {
			fmt.Printf("image %s already exists - please delete this first", imgName)
			os.Exit(1)
		}
	}

	_, err = imagesClient.CreateOrUpdate(context.TODO(), a.groupName, imgName, imageParams)
	if err != nil {
		log.Error(err)
	} else {
		log.Info("Image created")
	}

	return nil
}

// GetImages return all images for azure
func (a *Azure) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	var cimages []lepton.CloudImage

	imagesClient := a.getImagesClient()

	images, err := imagesClient.List(context.TODO())
	if err != nil {
		return nil, err
	}

	imgs := images.Values()

	for _, image := range imgs {
		if hasAzureOpsTags(image.Tags) {

			var t time.Time
			val, ok := image.Tags["CreatedAt"]
			if ok {
				t, err = time.Parse(time.RFC3339, *val)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				t = time.Now() // hack
			}

			cImage := lepton.CloudImage{
				Name:    *image.Name,
				Created: t,
				Status:  *(*image.ImageProperties).ProvisioningState,
			}

			cimages = append(cimages, cImage)
		}
	}

	return cimages, nil
}

// ListImages lists images on azure
func (a *Azure) ListImages(ctx *lepton.Context) error {

	cimages, err := a.GetImages(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range cimages {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, fmt.Sprintf("%v", image.Created))
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from Azure
func (a *Azure) DeleteImage(ctx *lepton.Context, imagename string) error {
	imagesClient := a.getImagesClient()

	_, err := imagesClient.Delete(context.TODO(), a.groupName, imagename)
	if err != nil {
		return err
	}

	err = a.Storage.DeleteFromBucket(ctx.Config(), imagename+".vhd")
	if err != nil {
		return err
	}

	return nil
}

// SyncImage syncs image from provider to another provider
func (a *Azure) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on azure.
func (a *Azure) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}
