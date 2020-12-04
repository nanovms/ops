package lepton

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/olekukonko/tablewriter"
)

// DigitalOcean provides access to the DigitalOcean API.
type DigitalOcean struct {
	Storage *Spaces
	Client  *godo.Client
}

// BuildImage to be upload on DO
func (do *DigitalOcean) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return do.customizeImage(ctx)
}

// BuildImageWithPackage to upload on DO .
func (do *DigitalOcean) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return do.customizeImage(ctx)
}

func (do *DigitalOcean) createImage(key string, bucket string, region string) {
	url := "https://api.digitalocean.com/v2/images"

	objURL := do.Storage.getSignedURL(key, bucket, region)

	token := os.Getenv("TOKEN")

	var jsonStr = []byte(`{"name": "` + key + `", "url": "` +
		objURL + `", "distribution": "Unknown", "region": "nyc3", "description":
 "` + key + `", "tags":["` + key + `"]}`)

	fmt.Println(string(jsonStr))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

// Initialize DigialOcean related things
func (do *DigitalOcean) Initialize(config *ProviderConfig) error {
	doToken := os.Getenv("TOKEN")
	do.Client = godo.NewFromToken(doToken)
	return nil
}

// CreateImage - Creates image on DO using nanos images
//
// This is currently blocked on DO allowing us to give it signed urls
// for image imports. If you wish to manage with digital ocean you need
// to manually upload the image for now.
//
// https://github.com/nanovms/ops/issues/468
func (do *DigitalOcean) CreateImage(ctx *Context) error {
	fmt.Println("Sorry - blocked on #468")
	os.Exit(1)

	c := ctx.config
	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName + ".img"
	zone := c.CloudConfig.Zone

	do.createImage(key, bucket, zone)

	return nil
}

// GetImages return all images on DigitalOcean
func (do *DigitalOcean) GetImages(ctx *Context) ([]CloudImage, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Images.List(context.TODO(), opt)
	if err != nil {
		return nil, err
	}
	images := make([]CloudImage, len(list))
	for i, doImage := range list {
		images[i].ID = fmt.Sprintf("%d", doImage.ID)
		images[i].Name = doImage.Name
		images[i].Status = doImage.Status
		images[i].Created = doImage.Created
	}
	return images, nil
}

// ListImages lists images on Digital Ocean.
func (do *DigitalOcean) ListImages(ctx *Context) error {
	images, err := do.GetImages(ctx)
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetRowLine(true)
	for _, image := range images {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, image.Created)
		table.Append(row)
	}
	table.Render()
	return nil
}

// DeleteImage deletes image from DO
func (do *DigitalOcean) DeleteImage(ctx *Context, imagename string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (do *DigitalOcean) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// ResizeImage is not supported on Digital Ocean.
func (do *DigitalOcean) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// CreateInstance - Creates instance on Digital Ocean Platform
func (do *DigitalOcean) CreateInstance(ctx *Context) error {
	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (do *DigitalOcean) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// GetInstances return all instances on DigitalOcean
// TODO
func (do *DigitalOcean) GetInstances(ctx *Context) ([]CloudInstance, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Droplets.List(context.TODO(), opt)
	if err != nil {
		return nil, err
	}
	cinstances := make([]CloudInstance, len(list))
	for i, droplet := range list {
		privateIPV4, _ := droplet.PrivateIPv4()
		publicIPV4, _ := droplet.PublicIPv4()
		publicIPV6, _ := droplet.PublicIPv6()
		cinstances[i] = CloudInstance{
			ID:         fmt.Sprintf("%d", droplet.ID),
			Name:       droplet.Name,
			Status:     droplet.Status,
			Created:    droplet.Created,
			PrivateIps: []string{privateIPV4},
			PublicIps:  []string{publicIPV4, publicIPV6},
		}
	}

	return cinstances, nil
}

// ListInstances lists instances on DO
func (do *DigitalOcean) ListInstances(ctx *Context) error {
	instances, err := do.GetInstances(ctx)
	if err != nil {
		return err
	}
	// print list of images in table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, instance := range instances {
		var rows []string
		rows = append(rows, instance.Name)
		rows = append(rows, instance.ID)
		rows = append(rows, instance.Status)
		rows = append(rows, instance.Created)
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))

		table.Append(rows)
	}
	table.Render()
	return nil

}

// DeleteInstance deletes instance from DO
func (do *DigitalOcean) DeleteInstance(ctx *Context, instancename string) error {
	return nil
}

// StartInstance starts an instance in DO
func (do *DigitalOcean) StartInstance(ctx *Context, instancename string) error {
	return nil
}

// StopInstance deletes instance from DO
func (do *DigitalOcean) StopInstance(ctx *Context, instancename string) error {
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (do *DigitalOcean) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := do.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (do *DigitalOcean) GetInstanceLogs(ctx *Context, instancename string) (string, error) {
	return "", nil
}

func (do *DigitalOcean) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() Storage {
	return do.Storage
}
