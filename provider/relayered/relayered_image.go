//go:build relayered || !onlyprovider

package relayered

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on relayered
func (v *relayered) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on relayered.
func (v *relayered) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.CustomizeImage(ctx)
}

func (v *relayered) destroyImage(snapshotid string) {
}

// CreateImage - Creates image on relayered using nanos images
func (v *relayered) CreateImage(ctx *lepton.Context, imagePath string) error {

	imgName := ctx.Config().CloudConfig.ImageName

	v.createImage(ctx, "", imgName)
	return nil
}

func (v *relayered) createImage(ctx *lepton.Context, icow string, imgName string) {
	filename := lepton.GetOpsHome() + "/" + "images/" + imgName

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", imgName)
	if err != nil {
		fmt.Println("error writing to buffer")
	}

	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		fmt.Println(err)
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	uri := baseURI + "/images/create"

	client := &http.Client{}
	req, err := http.NewRequest("POST", uri, bodyBuf)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", contentType)
	req.Header.Set("RELAYERED_TOKEN", v.token)
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

// ImageListResponse is the set of instances available from relayered in an
// images list call.
type ImageListResponse []Image

// Image represents a given relayered image configuration.
type Image struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetImages return all images on relayered
// needs tags added
func (v *relayered) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	client := &http.Client{}

	uri := baseURI + "/images/list"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("RELAYERED_TOKEN", v.token)
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

	var ilr ImageListResponse
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	var images []lepton.CloudImage

	for _, img := range ilr {
		images = append(images, lepton.CloudImage{
			ID:     img.ID,
			Name:   img.Name,
			Status: img.Status,
			Path:   "",
		})
	}

	return images, nil

}

// ListImages lists images on relayered
func (v *relayered) ListImages(ctx *lepton.Context) error {
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
func (v *relayered) DeleteImage(ctx *lepton.Context, snapshotID string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (v *relayered) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on relayered.
func (v *relayered) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *relayered) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}
