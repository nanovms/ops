package lepton

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Vultr provides access to the Vultr API.
type Vultr struct {
	Storage *Objects
}

// BuildImage to be upload on v
func (v *Vultr) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.customizeImage(ctx)
}

// BuildImageWithPackage to upload on Vultr.
func (v *Vultr) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.customizeImage(ctx)
}

func (v *Vultr) createImage(key string, bucket string, region string) {
	createURL := "https://api.vultr.com/v1/snapshot/create_from_url"

	objURL := v.Storage.getSignedURL(key, bucket, region)

	token := os.Getenv("TOKEN")

	urlData := url.Values{}
	urlData.Set("url", objURL)

	req, err := http.NewRequest("POST", createURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

// Initialize GCP related things
func (v *Vultr) Initialize() error {
	return nil
}

// CreateImage - Creates image on v using nanos images
func (v *Vultr) CreateImage(ctx *Context) error {

	c := ctx.config
	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName + ".img"
	zone := c.CloudConfig.Zone

	v.createImage(key, bucket, zone)

	return nil
}

type vultrSnap struct {
	SnapShotID  string `json:"SNAPSHOTID"`
	CreatedAt   string `json:"date_created"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Status      string `json:"status"`
	OSID        string `json:"OSID"`
	APPID       string `json:"APPID"`
}

// ListImages lists images on Digital Ocean
func (v *Vultr) ListImages(ctx *Context) error {

	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/snapshot/list", nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	token := os.Getenv("TOKEN")

	req.Header.Set("API-Key", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var data map[string]vultrSnap

	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println(err)
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
func (v *Vultr) DeleteImage(ctx *Context, imagename string) error {
	return nil
}

// CreateInstance - Creates instance on Digital Ocean Platform
func (v *Vultr) CreateInstance(ctx *Context) error {
	c := ctx.config

	// you may poll /v1/server/list?SUBID=<SUBID> and check that the "status" field is set to "active"

	createURL := "https://api.vultr.com/v1/server/create"

	token := os.Getenv("TOKEN")

	urlData := url.Values{}
	urlData.Set("DCID", "1")

	// this is the instance size
	// TODO
	urlData.Set("VPSPLANID", "201")

	// id for snapshot
	urlData.Set("OSID", "164")
	urlData.Set("SNAPSHOTID", c.CloudConfig.ImageName)

	req, err := http.NewRequest("POST", createURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	return nil
}

type vultrServer struct {
	SUBID     string `json:"SUBID"`
	Status    string `json:"status"`
	PublicIP  string `json:"main_ip"`
	PrivateIP string `json:"internal_ip"`
	CreatedAt string `json:"date_created"`
	Name      string `json:"label"`
}

// ListInstances lists instances on v
func (v *Vultr) ListInstances(ctx *Context) error {

	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/server/list", nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	token := os.Getenv("TOKEN")

	req.Header.Set("API-Key", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var data map[string]vultrServer

	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range data {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, image.CreatedAt)
		row = append(row, image.PrivateIP)
		row = append(row, image.PublicIP)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from v
func (v *Vultr) DeleteInstance(ctx *Context, instancename string) error {
	return nil
}

// StartInstance starts an instance in v
func (v *Vultr) StartInstance(ctx *Context, instancename string) error {
	return nil
}

// StopInstance deletes instance from v
func (v *Vultr) StopInstance(ctx *Context, instancename string) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Vultr) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {
	return nil
}

// TOv - make me shared
func (v *Vultr) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}
