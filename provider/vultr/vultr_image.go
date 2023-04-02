//go:build vultr || !onlyprovider

package vultr

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"github.com/vultr/govultr/v2"
)

// BuildImage to be upload on v
func (v *Vultr) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on Vultr.
func (v *Vultr) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.CustomizeImage(ctx)
}

func (v *Vultr) createImage(key string, bucket string, region string) {

	objURL := v.Storage.getSignedURL(key, bucket, region)

	snap, err := v.Client.Snapshot.CreateFromURL(context.TODO(), &govultr.SnapshotURLReq{
		URL: objURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("snapshot:", snap)
}

func (v *Vultr) destroyImage(snapshotid string) {
	err := v.Client.Snapshot.Delete(context.TODO(), snapshotid)
	if err != nil {
		log.Fatal(err)
	}
}

// CreateImage - Creates image on v using nanos images
func (v *Vultr) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := v.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	c := ctx.Config()
	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName
	zone := c.CloudConfig.Zone

	v.createImage(key, bucket, zone)

	return nil
}

// GetImages return all images on Vultr
func (v *Vultr) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	snaps, _, err := v.Client.Snapshot.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
	})
	if err != nil {
		return nil, err
	}

	var images []lepton.CloudImage

	for _, snap := range snaps {
		images = append(images, lepton.CloudImage{
			ID:      snap.ID,
			Name:    "",
			Status:  snap.Status,
			Size:    int64(snap.Size),
			Path:    "",
			Created: time.Now(),
		})
	}

	return images, nil

}

// ListImages lists images on Vultr
func (v *Vultr) ListImages(ctx *lepton.Context) error {

	snaps, _, err := v.Client.Snapshot.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
	})
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Date created", "Size", "Status"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range snaps {
		var row []string
		row = append(row, image.ID)
		row = append(row, image.DateCreated)
		row = append(row, humanize.Bytes(uint64(image.Size)))
		row = append(row, image.Status)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from v
func (v *Vultr) DeleteImage(ctx *lepton.Context, snapshotID string) error {
	v.destroyImage(snapshotID)

	return nil
}

// SyncImage syncs image from provider to another provider
func (v *Vultr) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on Vultr.
func (v *Vultr) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *Vultr) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}
