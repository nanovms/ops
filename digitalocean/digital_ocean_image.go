package digitalocean

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

const (
	opsTag = "createdBy:ops"
)

// CreateImage - Creates image on DO using nanos images
//
// This is currently blocked on DO allowing us to give it signed urls
// for image imports. If you wish to manage with digital ocean you need
// to manually upload the image for now.
//
// https://github.com/nanovms/ops/issues/468
func (do *DigitalOcean) CreateImage(ctx *lepton.Context, imagePath string) error {
	c := ctx.Config()

	imageName := c.CloudConfig.ImageName + ".img"

	err := do.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	publicURL := do.Storage.getImageSpacesURL(c, c.CloudConfig.ImageName+".img")

	createImageRequest := &godo.CustomImageCreateRequest{
		Name:         imageName,
		Url:          publicURL,
		Distribution: "Unknown",
		Region:       ctx.Config().CloudConfig.Zone,
		Tags:         []string{opsTag},
	}

	_, _, err = do.Client.Images.Create(context.TODO(), createImageRequest)
	if err != nil {
		return err
	}

	err = do.Storage.DeleteFromBucket(c, filepath.Base(imagePath))
	if err != nil {
		return err
	}

	return nil
}

// GetImages return all images on DigitalOcean
func (do *DigitalOcean) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Images.ListByTag(context.TODO(), opsTag, opt)
	if err != nil {
		return nil, err
	}
	images := make([]lepton.CloudImage, len(list))
	for i, doImage := range list {
		images[i].ID = fmt.Sprintf("%d", doImage.ID)
		images[i].Name = doImage.Name
		images[i].Status = doImage.Status
		images[i].Created, _ = time.Parse("2006-01-02T15:04:05Z", doImage.Created)
	}
	return images, nil
}

// ListImages lists images on Digital Ocean.
func (do *DigitalOcean) ListImages(ctx *lepton.Context) error {
	images, err := do.GetImages(ctx)
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetRowLine(true)
	for _, image := range images {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, lepton.Time2Human(image.Created))
		table.Append(row)
	}
	table.Render()
	return nil
}

// DeleteImage deletes image from DO
func (do *DigitalOcean) DeleteImage(ctx *lepton.Context, imagename string) error {
	image, err := do.getImageByName(ctx, imagename)
	if err != nil {
		return err
	}

	id, _ := strconv.Atoi(image.ID)

	_, err = do.Client.Images.Delete(context.TODO(), id)
	if err != nil {
		return err
	}

	return nil
}

// SyncImage syncs image from provider to another provider
func (do *DigitalOcean) SyncImage(config *lepton.Config, target lepton.Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// ResizeImage is not supported on Digital Ocean.
func (do *DigitalOcean) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (do *DigitalOcean) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.Imagename
	return imagePath, nil
}

// BuildImage to be upload on DO
func (do *DigitalOcean) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return do.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on DO .
func (do *DigitalOcean) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return do.CustomizeImage(ctx)
}

func (do *DigitalOcean) getImageByName(ctx *lepton.Context, imageName string) (*lepton.CloudImage, error) {
	images, err := do.GetImages(ctx)
	if err != nil {
		return nil, err
	}

	var image *lepton.CloudImage
	for _, i := range images {
		if i.Name == imageName {
			image = &i
		}
	}

	if image == nil {
		return nil, fmt.Errorf(`image with name "%s" not found`, imageName)
	}

	return image, nil
}
