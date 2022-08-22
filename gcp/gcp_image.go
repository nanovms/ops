package gcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// GCPStorageURL is GCP storage path
const GCPStorageURL string = "https://storage.googleapis.com/%v/%v"

func amendConfig(c *types.Config) {
	if strings.HasPrefix(c.CloudConfig.Flavor, "t2a") {
		c.Uefi = true
	}
}

// BuildImage to be upload on GCP
func (p *GCloud) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	amendConfig(c)
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.CustomizeImage(ctx)
}

func (p *GCloud) getArchiveName(ctx *lepton.Context) string {
	return ctx.Config().CloudConfig.ImageName + ".tar.gz"
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (p *GCloud) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.Imagename
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

	archPath := filepath.Join(filepath.Dir(imagePath), p.getArchiveName(ctx))
	files := []string{symlink}

	err = lepton.CreateArchive(archPath, files)
	if err != nil {
		return "", fmt.Errorf("failed creating archive: %v", err)
	}
	return archPath, nil
}

// BuildImageWithPackage to upload on GCP
func (p *GCloud) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	amendConfig(c)
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.CustomizeImage(ctx)
}

// CreateImage - Creates image on GCP using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *GCloud) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := p.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}
	defer func() {
		p.Storage.DeleteFromBucket(ctx.Config(), imagePath)
		defer os.Remove(imagePath)
	}()

	c := ctx.Config()
	context := context.TODO()

	sourceURL := fmt.Sprintf(GCPStorageURL,
		c.CloudConfig.BucketName, p.getArchiveName(ctx))

	rb := &compute.Image{
		Name:   c.CloudConfig.ImageName,
		Labels: buildGcpTags(ctx.Config().CloudConfig.Tags),
		RawDisk: &compute.ImageRawDisk{
			Source: sourceURL,
		},
	}

	op, err := p.Service.Images.Insert(c.CloudConfig.ProjectID, rb).Context(context).Do()
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	fmt.Printf("Image creation started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, c.CloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Image creation succeeded %s.\n", c.CloudConfig.ImageName)
	return nil
}

// GetImages return all images on GCloud
func (p *GCloud) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return nil, err
	}
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	var images []lepton.CloudImage

	req := computeService.Images.List(creds.ProjectID)
	err = req.Pages(context, func(page *compute.ImageList) error {
		for _, image := range page.Items {
			if val, ok := image.Labels["createdby"]; ok && val == "ops" {
				imageCreatedAt, _ := time.Parse("2006-01-02T15:04:05-07:00", image.CreationTimestamp)

				ci := lepton.CloudImage{
					Name:    image.Name,
					Status:  fmt.Sprintf("%v", image.Status),
					Created: imageCreatedAt,
				}

				images = append(images, ci)
			}
		}
		return nil
	})

	return images, err

}

// ListImages lists images on Google Cloud
func (p *GCloud) ListImages(ctx *lepton.Context) error {
	images, err := p.GetImages(ctx)
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

// DeleteImage deletes image from Gcloud
func (p *GCloud) DeleteImage(ctx *lepton.Context, imagename string) error {
	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return err
	}
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return err
	}
	op, err := computeService.Images.Delete(creds.ProjectID, imagename).Context(context).Do()
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	err = p.pollOperation(context, creds.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Image deletion succeeded %s.\n", imagename)
	return nil
}

// SyncImage syncs image from provider to another provider
func (p *GCloud) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on google cloud.
func (p *GCloud) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}
