//go:build digitalocean || do || !onlyprovider

package digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

const (
	opsTag = "createdBy:ops"
)

// CreateImage - Creates image on DO using nanos images
// converts to qcow2 first
func (do *DigitalOcean) CreateImage(ctx *lepton.Context, imagePath string) error {
	c := ctx.Config()

	imageName := c.CloudConfig.ImageName
	newPath := c.CloudConfig.ImageName + ".qcow"

	cmd := exec.Command("sh", "-c", "qemu-img convert -f raw -O qcow2 ~/.ops/images/"+imageName+" ~/.ops/images/"+newPath)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Printf("%s\n", stdoutStderr)
		return err
	}

	opshome := lepton.GetOpsHome()

	err = do.Storage.CopyToBucket(ctx.Config(), opshome+"/images/"+newPath)
	if err != nil {
		return err
	}

	publicURL := do.Storage.getImageSpacesURL(c, newPath)
	publicURL = do.Storage.getSignedURL(newPath, c.CloudConfig.BucketName, c.CloudConfig.Zone)

	log.Info(publicURL)

	createImageRequest := &godo.CustomImageCreateRequest{
		Name:         imageName,
		Url:          publicURL,
		Distribution: "Unknown",
		Region:       ctx.Config().CloudConfig.Zone,
		Tags:         []string{opsTag},
	}

	i, _, err := do.Client.Images.Create(context.TODO(), createImageRequest)
	if err != nil {
		return err
	}

	log.Infof("%+v\n", i)

	for i.Status != "available" {
		time.Sleep(250 * time.Millisecond)
		i, _, err = do.Client.Images.GetByID(context.TODO(), i.ID)
		if err != nil {
			return err
		}
	}

	err = do.Storage.DeleteFromBucket(c, filepath.Base(newPath))
	if err != nil {
		return err
	}

	return nil
}

// GetImages return all images on DigitalOcean
func (do *DigitalOcean) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
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
func (do *DigitalOcean) ListImages(ctx *lepton.Context, filter string) error {
	images, err := do.GetImages(ctx, "")
	if err != nil {
		return err
	}
	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(images)
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
func (do *DigitalOcean) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on Digital Ocean.
func (do *DigitalOcean) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (do *DigitalOcean) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
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
	images, err := do.GetImages(ctx, "")
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
