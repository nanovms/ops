package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/go-errors/errors"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

// UnWarpConfig parses lepton config file from file
func unWarpConfig(file string) *api.Config {
	var c api.Config
	if file != "" {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(data, &c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error config: %v\n", err)
			os.Exit(1)
		}
	}
	return &c
}

// SetDefaultImageName set default name for an image
func setDefaultImageName(cmd *cobra.Command, c *api.Config) {
	// if user have not supplied an imagename, use the default as program_image
	// all images goes to $HOME/.ops/images
	imageName, _ := cmd.Flags().GetString("imagename")
	if imageName == "" {
		imageName = api.GenerateImageName(c.Program)
		c.CloudConfig.ImageName = fmt.Sprintf("%v-image", filepath.Base(c.Program))
	} else {
		c.CloudConfig.ImageName = imageName
		images := path.Join(api.GetOpsHome(), "images")
		imageName = path.Join(images, filepath.Base(imageName))
	}
	c.RunConfig.Imagename = imageName
}

// TODO : use factory or DI
func getCloudProvider(providerName string) api.Provider {
	var provider api.Provider

	if providerName == "gcp" {
		provider = &api.GCloud{}
	} else if providerName == "onprem" {
		provider = &api.OnPrem{}
	} else if providerName == "aws" {
		provider = &api.AWS{}
	} else if providerName == "do" {
		provider = &api.DigitalOcean{}
	} else if providerName == "vultr" {
		provider = &api.Vultr{}
	} else if providerName == "vsphere" {
		provider = &api.Vsphere{}
	} else if providerName == "azure" {
		provider = &api.Azure{}
	} else {
		fmt.Fprintf(os.Stderr, "error:Unknown provider %s", providerName)
		os.Exit(1)
	}

	provider.Initialize()
	return provider
}

func initDefaultRunConfigs(c *api.Config, ports []int) {

	if c.RunConfig.Memory == "" {
		c.RunConfig.Memory = "2G"
	}
	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)
}

func fixupConfigImages(c *api.Config, version string) {
	if c.NightlyBuild {
		version = "nightly"
	}

	if c.Boot == "" {
		c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	}

	if c.Kernel == "" {
		c.Kernel = path.Join(api.GetOpsHome(), version, "stage3.img")
	}

	if c.Mkfs == "" {
		c.Mkfs = path.Join(api.GetOpsHome(), version, "mkfs")
	}

	if c.NameServer == "" {
		// google dns server
		c.NameServer = "8.8.8.8"
	}
}

func validateRequired(c *api.Config) {
	if _, err := os.Stat(c.Kernel); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Kernel, err)
		os.Exit(1)
	}
	if _, err := os.Stat(c.Mkfs); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Mkfs, err)
		os.Exit(1)
	}
	if _, err := os.Stat(c.Boot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Boot, err)
		os.Exit(1)
	}
	_, err := os.Stat(path.Join(api.GetOpsHome(), c.Program))
	_, err1 := os.Stat(c.Program)

	if os.IsNotExist(err) && os.IsNotExist(err1) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Program, err)
		os.Exit(1)
	}
}

func prepareImages(c *api.Config) {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else {
		currversion, err = downloadReleaseImages()
	}

	panicOnError(err)
	fixupConfigImages(c, currversion)
	validateRequired(c)
}

func panicOnError(err error) {
	if err != nil {
		fmt.Println(err.(*errors.Error).ErrorStack())
		panic(err)
	}
}

func downloadAndExtractPackage(pkg string) string {
	localstaging := path.Join(api.GetOpsHome(), ".staging")
	err := os.MkdirAll(localstaging, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	expackage := path.Join(localstaging, pkg)
	localpackage, err := api.DownloadPackage(pkg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Remove the folder first.
	os.RemoveAll(expackage)
	api.ExtractPackage(localpackage, localstaging)
	return expackage
}
