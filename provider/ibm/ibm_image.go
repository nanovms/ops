//go:build ibm || !onlyprovider

package ibm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on IBM
func (v *IBM) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on IBM.
func (v *IBM) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.CustomizeImage(ctx)
}

func (v *IBM) destroyImage(snapshotid string) {
}

// CreateImage - Creates image on IBM using nanos images
func (v *IBM) CreateImage(ctx *lepton.Context, imagePath string) error {
	// also worth gzipping

	icow := imagePath + ".qcow2"

	args := []string{
		"convert", "-f", "raw", "-O", "qcow2", imagePath, icow,
	}

	cmd := exec.Command("qemu-img", args...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		fmt.Println(out)
	}

	store := &Objects{
		token: v.iam,
	}

	v.Storage = store
	err = v.Storage.CopyToBucket(ctx.Config(), icow)
	if err != nil {
		return err
	}

	imgName := ctx.Config().CloudConfig.ImageName

	v.createImage(ctx, icow, imgName)
	return nil
}

func (v *IBM) createImage(ctx *lepton.Context, icow string, imgName string) {
	baseName := filepath.Base(icow)

	c := ctx.Config()
	zone := c.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	bucket := c.CloudConfig.BucketName

	uri := "https://" + region + ".iaas.cloud.ibm.com/v1/images?version=2023-02-16&generation=2"

	rgroup := v.getDefaultResourceGroup()

	j := `{
	     "name": "` + imgName + `",
	     "operating_system": {
	       "name": "ubuntu-18-04-amd64"
	     },
	     "file": {
	       "href": "cos://` + region + `/` + bucket + `/` + baseName + `"
	     },
	     "resource_group": {
	       "id": "` + rgroup + `"
	     }
	   }`

	reqBody := []byte(j)

	client := &http.Client{}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(body))
}

// ImageListResponse is the set of instances available from IBM in an
// images list call.
type ImageListResponse struct {
	Images []Image `json:"images"`
}

// Image represents a given IBM image configuration.
type Image struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetImages return all images on IBM
// needs tags added
func (v *IBM) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	client := &http.Client{}

	c := ctx.Config()
	zone := c.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	uri := "https://" + region + ".iaas.cloud.ibm.com/v1/images?version=2023-02-28&generation=2&visibility=private"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)

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

	for _, img := range ilr.Images {
		images = append(images, lepton.CloudImage{
			ID:     img.ID,
			Name:   img.Name,
			Status: img.Status,
			Path:   "",
		})
	}

	return images, nil

}

// ListImages lists images on IBM
func (v *IBM) ListImages(ctx *lepton.Context) error {
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
		row = append(row, "")
		row = append(row, humanize.Bytes(uint64(image.Size)))
		row = append(row, image.Status)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from v
func (v *IBM) DeleteImage(ctx *lepton.Context, snapshotID string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (v *IBM) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on IBM.
func (v *IBM) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *IBM) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}
