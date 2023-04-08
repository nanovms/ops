//go:build linode || !onlyprovider

package linode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on linode
func (v *Linode) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on Linode.
func (v *Linode) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.CustomizeImage(ctx)
}

func (v *Linode) createImage(key string, bucket string, region string) {
}

func (v *Linode) destroyImage(snapshotid string) {
}

// CreateImage - Creates image on linode using nanos images
func (v *Linode) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := v.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	return nil
}

// ImageListResponse is the set of instances available from linode in an
// images list call.
type ImageListResponse struct {
	Data []Image `json:"data"`
}

// Image represents a given linode image configuration.
type Image struct {
	CreatedBy string `json:"created_by"`
	ID        string `json:"id"`
	IsPublic  bool   `json:"is_public"`
	Label     string `json:"label"`
	Status    string `json:"status"`
}

// GetImages return all images on Linode
func (v *Linode) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	token := os.Getenv("TOKEN")
	client := &http.Client{}

	uri := "https://api.linode.com/v4/images"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ilr := &ImageListResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	var images []lepton.CloudImage

	for _, img := range ilr.Data {
		if !img.IsPublic {
			images = append(images, lepton.CloudImage{
				ID:     img.ID,
				Name:   img.Label,
				Status: img.Status,
				Path:   "",
			})
		}
	}

	return images, nil

}

// ListImages lists images on Linode
func (v *Linode) ListImages(ctx *lepton.Context) error {
	images, err := v.GetImages(ctx)
	if err != nil {
		fmt.Println(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Date created", "Size", "Status"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range images {
		var row []string
		row = append(row, image.ID)
		row = append(row, image.Name)
		row = append(row, "") // image.DateCreated)
		row = append(row, humanize.Bytes(uint64(image.Size)))
		row = append(row, image.Status)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from v
func (v *Linode) DeleteImage(ctx *lepton.Context, snapshotID string) error {
	//	v.destroyImage(snapshotID)

	return nil
}

// SyncImage syncs image from provider to another provider
func (v *Linode) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on Linode.
func (v *Linode) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *Linode) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}
