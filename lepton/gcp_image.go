package lepton

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

func (p *GCloud) getArchiveName(ctx *Context) string {
	return ctx.config.CloudConfig.ImageName + ".tar.gz"
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (p *GCloud) CustomizeImage(ctx *Context) (string, error) {
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

	archPath := filepath.Join(filepath.Dir(imagePath), p.getArchiveName(ctx))
	files := []string{symlink}

	err = createArchive(archPath, files)
	if err != nil {
		return "", err
	}
	return archPath, nil
}

// BuildImage to be upload on GCP
func (p *GCloud) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on GCP
func (p *GCloud) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.CustomizeImage(ctx)
}

// CreateImage - Creates image on GCP using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *GCloud) CreateImage(ctx *Context, imagePath string) error {
	err := p.Storage.CopyToBucket(ctx.config, imagePath)
	if err != nil {
		return err
	}

	c := ctx.config
	context := context.TODO()

	sourceURL := fmt.Sprintf(GCPStorageURL,
		c.CloudConfig.BucketName, p.getArchiveName(ctx))

	rb := &compute.Image{
		Name:   c.CloudConfig.ImageName,
		Labels: buildGcpTags(ctx.config.RunConfig.Tags),
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
func (p *GCloud) GetImages(ctx *Context) ([]CloudImage, error) {
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

	var images []CloudImage

	req := computeService.Images.List(creds.ProjectID)
	err = req.Pages(context, func(page *compute.ImageList) error {
		for _, image := range page.Items {
			if val, ok := image.Labels["createdby"]; ok && val == "ops" {
				imageCreatedAt, _ := time.Parse("2006-01-02T15:04:05-07:00", image.CreationTimestamp)

				ci := CloudImage{
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
func (p *GCloud) ListImages(ctx *Context) error {
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
		row = append(row, Time2Human(image.Created))
		table.Append(row)
	}

	table.Render()
	return nil
}

// DeleteImage deletes image from Gcloud
func (p *GCloud) DeleteImage(ctx *Context, imagename string) error {
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
func (p *GCloud) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

func createArchive(archive string, files []string) error {
	fd, err := os.Create(archive)
	if err != nil {
		return err
	}
	gzw := gzip.NewWriter(fd)

	tw := tar.NewWriter(gzw)

	for _, file := range files {
		fstat, err := os.Stat(file)
		if err != nil {
			return err
		}

		// write the header
		if err := tw.WriteHeader(&tar.Header{
			Name:   filepath.Base(file),
			Mode:   int64(fstat.Mode()),
			Size:   fstat.Size(),
			Format: tar.FormatGNU,
		}); err != nil {
			return err
		}

		fi, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data to tar
		if _, err := io.CopyN(tw, fi, fstat.Size()); err != nil {
			return err
		}
		if err = fi.Close(); err != nil {
			return err
		}
	}

	// Explicitly close all writers in correct order without any error
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	return nil
}

// ResizeImage is not supported on google cloud.
func (p *GCloud) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}
