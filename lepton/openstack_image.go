package lepton

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/olekukonko/tablewriter"
)

// ResizeImage is not supported on OpenStack.
func (o *OpenStack) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// BuildImage to be upload on OpenStack
func (o *OpenStack) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return o.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on OpenStack.
func (o *OpenStack) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return o.CustomizeImage(ctx)
}

func (o *OpenStack) findImage(name string) (id string, err error) {

	imageClient, err := openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println(err)
	}

	listOpts := images.ListOpts{
		Name: name,
	}

	allPages, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		panic(err)
	}

	// yolo
	// names are not unique so this is just (for now) grabbing first
	// result
	// FIXME
	if len(allImages) > 0 {
		return allImages[0].ID, nil
	}

	return "", errors.New("not found")
}

func (o *OpenStack) getImagesClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}

func (o *OpenStack) createImage(imagesClient *gophercloud.ServiceClient, imgName string) (*images.Image, error) {
	visibility := images.ImageVisibilityPrivate

	createOpts := images.CreateOpts{
		Name:            imgName,
		DiskFormat:      "raw",
		ContainerFormat: "bare",
		Visibility:      &visibility,
		Tags:            getOpenstackOpsTags(),
	}

	return images.Create(imagesClient, createOpts).Extract()
}

func (o *OpenStack) uploadImage(imagesClient *gophercloud.ServiceClient, imageID string, imagePath string) error {
	imageData, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imageData.Close()

	return imagedata.Upload(imagesClient, imageID, imageData).ExtractErr()
}

// CreateImage - Creates image on OpenStack using nanos images
func (o *OpenStack) CreateImage(ctx *Context, imagePath string) error {
	c := ctx.config
	imgName := c.CloudConfig.ImageName

	imgName = strings.ReplaceAll(imgName, "-image", "")

	fmt.Println("creating image:\t" + imgName)

	imagesClient, err := o.getImagesClient()
	if err != nil {
		fmt.Println(err)
	}

	image, err := o.createImage(imagesClient, imgName)
	if err != nil {
		fmt.Println(err)
	}

	imagePath = localImageDir + "/" + imgName
	err = o.uploadImage(imagesClient, image.ID, imagePath)
	if err != nil {
		return err
	}

	return nil
}

// GetImages return all images for openstack
func (o *OpenStack) GetImages(ctx *Context) ([]CloudImage, error) {
	var cimages []CloudImage

	imageClient, err := openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println(err)
	}

	listOpts := images.ListOpts{
		Tags: getOpenstackOpsTags(),
	}

	allPages, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		fmt.Println(err)
	}

	for _, image := range allImages {

		cimage := CloudImage{
			Name:    image.Name,
			Status:  string(image.Status),
			Created: image.CreatedAt,
		}

		cimages = append(cimages, cimage)
	}

	return cimages, nil
}

// ListImages lists images on a datastore.
func (o *OpenStack) ListImages(ctx *Context) error {

	cimages, err := o.GetImages(ctx)
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
		row = append(row, Time2Human(image.Created))

		table.Append(row)
	}

	table.Render()

	return nil
}

func (o *OpenStack) deleteImage(imagesClient *gophercloud.ServiceClient, imageID string) error {
	return images.Delete(imagesClient, imageID).ExtractErr()
}

// DeleteImage deletes image from OpenStack
func (o *OpenStack) DeleteImage(ctx *Context, imagename string) error {
	imageID, err := o.findImage(imagename)
	if err != nil {
		fmt.Println(err)
		return err
	}

	imageClient, err := o.getImagesClient()
	if err != nil {
		fmt.Println(err)
	}

	err = images.Delete(imageClient, imageID).ExtractErr()
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// SyncImage syncs image from provider to another provider
func (o *OpenStack) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (o *OpenStack) CustomizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}
