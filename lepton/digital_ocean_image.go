package lepton

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/digitalocean/godo"
	"github.com/olekukonko/tablewriter"
)

// CreateImage - Creates image on DO using nanos images
//
// This is currently blocked on DO allowing us to give it signed urls
// for image imports. If you wish to manage with digital ocean you need
// to manually upload the image for now.
//
// https://github.com/nanovms/ops/issues/468
func (do *DigitalOcean) CreateImage(ctx *Context, imagePath string) error {
	err := do.Storage.CopyToBucket(ctx.config, imagePath)
	if err != nil {
		return err
	}

	fmt.Println("Sorry - blocked on #468")
	os.Exit(1)

	c := ctx.config
	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName + ".img"
	zone := c.CloudConfig.Zone

	do.createImage(key, bucket, zone)

	return nil
}

// GetImages return all images on DigitalOcean
func (do *DigitalOcean) GetImages(ctx *Context) ([]CloudImage, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Images.List(context.TODO(), opt)
	if err != nil {
		return nil, err
	}
	images := make([]CloudImage, len(list))
	for i, doImage := range list {
		images[i].ID = fmt.Sprintf("%d", doImage.ID)
		images[i].Name = doImage.Name
		images[i].Status = doImage.Status
		images[i].Created, _ = time.Parse("2006-01-02T15:04:05Z", doImage.Created)
	}
	return images, nil
}

// ListImages lists images on Digital Ocean.
func (do *DigitalOcean) ListImages(ctx *Context) error {
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
		row = append(row, time2Human(image.Created))
		table.Append(row)
	}
	table.Render()
	return nil
}

// DeleteImage deletes image from DO
func (do *DigitalOcean) DeleteImage(ctx *Context, imagename string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (do *DigitalOcean) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// ResizeImage is not supported on Digital Ocean.
func (do *DigitalOcean) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// GetInstanceLogs gets instance related logs
func (do *DigitalOcean) GetInstanceLogs(ctx *Context, instancename string) (string, error) {
	return "", nil
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (do *DigitalOcean) CustomizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}
