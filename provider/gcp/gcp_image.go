//go:build gcp || !onlyprovider

package gcp

import (
	"context"
	"encoding/json"
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

func getArchitecture(flavor string) string {
	if flavor != "" {
		armFlavors := []string{"t2a", "c4a"}
		for _, armFlavor := range armFlavors {
			if strings.HasPrefix(flavor, armFlavor) {
				return "arm64"
			}
		}
	}
	return "x86_64"
}

func amendConfig(c *types.Config) {
	if getArchitecture(c.CloudConfig.Flavor) == "arm64" {
		c.Uefi = true
	}
	if c.CloudConfig.ConfidentialVM {
		/* Confidential VM feature can only be enabled with UEFI-compatible images */
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
	imagePath := ctx.Config().RunConfig.ImageName
	archPath := filepath.Join(filepath.Dir(imagePath), p.getArchiveName(ctx))
	files := map[string]string{
		imagePath: "disk.raw",
	}
	err := lepton.CreateArchive(archPath, files)
	if err != nil {
		return "", fmt.Errorf("failed creating archive: %w", err)
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
		Name:         c.CloudConfig.ImageName,
		Labels:       buildGcpLabels(ctx.Config().CloudConfig.Tags, "image"),
		Architecture: "X86_64",
		GuestOsFeatures: []*compute.GuestOsFeature{
			{
				Type: "GVNIC",
			},
		},
		RawDisk: &compute.ImageRawDisk{
			Source: sourceURL,
		},
	}

	if c.Uefi {
		rb.GuestOsFeatures = append(rb.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "UEFI_COMPATIBLE",
		})
	}

	if getArchitecture(c.CloudConfig.Flavor) == "arm64" {
		rb.Architecture = "ARM64"
	} else {
		rb.GuestOsFeatures = append(rb.GuestOsFeatures, &compute.GuestOsFeature{
			Type: "SEV_CAPABLE",
		})
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
func (p *GCloud) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
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

				labels := []string{}
				for k, v := range image.Labels {
					labels = append(labels, k+":"+v)
				}

				ci := lepton.CloudImage{
					Name:    image.Name,
					Status:  fmt.Sprintf("%v", image.Status),
					Created: imageCreatedAt,
					Labels:  labels,
				}

				images = append(images, ci)
			}
		}
		return nil
	})

	return images, err

}

// ListImages lists images on Google Cloud
func (p *GCloud) ListImages(ctx *lepton.Context, filter string) error {
	images, err := p.GetImages(ctx, filter)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(images)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Labels"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range images {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, lepton.Time2Human(image.Created))
		row = append(row, strings.Join(image.Labels[:], ","))
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
