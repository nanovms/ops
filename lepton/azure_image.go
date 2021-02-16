package lepton

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/config"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on Azure
func (a *Azure) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return a.CustomizeImage(ctx)
}

func (a *Azure) getArchiveName(ctx *Context) string {
	return ctx.config.CloudConfig.ImageName + ".tar.gz"
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (a *Azure) CustomizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
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

	err = createArchive(archPath, files)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// BuildImageWithPackage to upload on Azure
func (a *Azure) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return a.CustomizeImage(ctx)
}

// CreateImage - Creates image on Azure using nanos images
func (a *Azure) CreateImage(ctx *Context, imagePath string) error {
	err := a.Storage.CopyToBucket(ctx.config, imagePath)
	if err != nil {
		return err
	}

	imagesClient := a.getImagesClient()

	c := ctx.config
	imgName := c.CloudConfig.ImageName

	bucket, err := a.getBucketName()
	if err != nil {
		return err
	}

	region := a.getLocation(ctx.config)
	container := "quickstart-nanos"
	disk := c.CloudConfig.ImageName + ".vhd"

	uri := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	imageParams := compute.Image{
		Location: to.StringPtr(region),
		Tags:     getAzureDefaultTags(),
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					OsType:  compute.Linux,
					BlobURI: to.StringPtr(uri),
					OsState: compute.Generalized,
				},
			},
			HyperVGeneration: compute.HyperVGenerationTypesV1,
		},
	}

	_, err = imagesClient.CreateOrUpdate(context.TODO(), a.groupName, imgName, imageParams)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Image created")
	}

	return nil
}

// GetImages return all images for azure
func (a *Azure) GetImages(ctx *Context) ([]CloudImage, error) {
	var cimages []CloudImage

	imagesClient := a.getImagesClient()

	images, err := imagesClient.List(context.TODO())
	if err != nil {
		return nil, err
	}

	imgs := images.Values()

	for _, image := range imgs {
		if hasAzureOpsTags(image.Tags) {
			cImage := CloudImage{
				Name:   *image.Name,
				Status: *(*image.ImageProperties).ProvisioningState,
			}

			cimages = append(cimages, cImage)
		}
	}

	return cimages, nil
}

// ListImages lists images on azure
func (a *Azure) ListImages(ctx *Context) error {

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
		row = append(row, "")
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from Azure
func (a *Azure) DeleteImage(ctx *Context, imagename string) error {
	imagesClient := a.getImagesClient()

	_, err := imagesClient.Delete(context.TODO(), a.groupName, imagename)
	if err != nil {
		return err
	}

	err = a.Storage.DeleteFromBucket(ctx.config, imagename+".vhd")
	if err != nil {
		return err
	}

	return nil
}

// SyncImage syncs image from provider to another provider
func (a *Azure) SyncImage(config *config.Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// ResizeImage is not supported on azure.
func (a *Azure) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}
