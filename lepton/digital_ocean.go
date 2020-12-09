package lepton

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/digitalocean/godo"
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

	return do.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on DO .
func (do *DigitalOcean) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return do.CustomizeImage(ctx)
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

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() Storage {
	return do.Storage
}
