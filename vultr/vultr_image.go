package vultr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
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
	createURL := "https://api.vultr.com/v1/snapshot/create_from_url"

	objURL := v.Storage.getSignedURL(key, bucket, region)

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("url", objURL)

	req, err := http.NewRequest("POST", createURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))
}

func (v *Vultr) destroyImage(snapshotid string) {
	destroyURL := "https://api.vultr.com/v1/snapshot/destroy"

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("SNAPSHOTID", snapshotid)

	req, err := http.NewRequest("POST", destroyURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))
}

// CreateImage - Creates image on v using nanos images
func (v *Vultr) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := v.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	c := ctx.Config()
	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName + ".img"
	zone := c.CloudConfig.Zone

	v.createImage(key, bucket, zone)

	return nil
}

// GetImages return all images on Vultr
func (v *Vultr) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, errors.New("un-implemented")
}

// ListImages lists images on Digital Ocean
func (v *Vultr) ListImages(ctx *lepton.Context) error {

	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/snapshot/list", nil)
	if err != nil {
		log.Fatal(err)
	}
	token := os.Getenv("VULTR_TOKEN")

	req.Header.Set("API-Key", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data map[string]vultrSnap

	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Error(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range data {
		var row []string
		row = append(row, image.Description)
		row = append(row, image.SnapShotID)
		row = append(row, image.Status)
		row = append(row, image.CreatedAt)
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
	return fmt.Errorf("Operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *Vultr) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.Imagename
	return imagePath, nil
}
