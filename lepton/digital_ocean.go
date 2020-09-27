package lepton

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// DigitalOcean provides access to the DigitalOcean API.
type DigitalOcean struct {
	Storage *Spaces
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

// Initialize GCP related things
func (do *DigitalOcean) Initialize() error {
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
	return nil, errors.New("un-implemented")
}

// ListImages lists images on Digital Ocean
func (do *DigitalOcean) ListImages(ctx *Context) error {

	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/images?page=1&per_page=1", nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	token := os.Getenv("TOKEN")

	req.Header.Set("Authorization", "Bearer "+token)
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
	fmt.Println(body)

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

// GetInstances return all instances on DigitalOcean
// TODO
func (do *DigitalOcean) GetInstances(ctx *Context) ([]CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// ListInstances lists instances on DO
func (do *DigitalOcean) ListInstances(ctx *Context) error {
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

// GetInstanceLogs gets instance related logs
func (do *DigitalOcean) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {
	return nil
}

func (do *DigitalOcean) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() Storage {
	return do.Storage
}
