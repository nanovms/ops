package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/hyperv"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/upcloud"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	os.Exit(1)
}

func exitForCmd(cmd *cobra.Command, errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	cmd.Help()
	os.Exit(1)
}

// unWarpConfig parses lepton config file from file
func unWarpConfig(file string) *api.Config {
	var c api.Config
	if file != "" {
		c = *api.NewConfig()
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
		return &c
	}
	c = *unWarpDefaultConfig()
	return &c
}

// unWarpDefaultConfig gets default config file from env
func unWarpDefaultConfig() *api.Config {
	c := *api.NewConfig()
	conf := os.Getenv("OPS_DEFAULT_CONFIG")
	if conf != "" {
		data, err := ioutil.ReadFile(conf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(data, &c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error config: %v\n", err)
			os.Exit(1)
		}
		return &c
	}
	usr, err := user.Current()
	if err != nil {
		return &c
	}
	conf = usr.HomeDir + "/.opsrc"
	_, err = os.Stat(conf)
	if err != nil {
		return &c
	}
	data, err := ioutil.ReadFile(conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error config: %v\n", err)
		os.Exit(1)
	}
	return &c
}

// setDefaultImageName set default name for an image
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
func getCloudProvider(providerName string, config *api.ProviderConfig) (api.Provider, error) {
	var provider api.Provider

	switch providerName {
	case "gcp":
		provider = api.NewGCloud()
	case "onprem":
		provider = &api.OnPrem{}
	case "aws":
		provider = &api.AWS{}
	case "do":
		provider = &api.DigitalOcean{}
	case "vultr":
		provider = &api.Vultr{}
	case "vsphere":
		provider = &api.Vsphere{}
	case "openstack":
		provider = &api.OpenStack{}
	case "azure":
		provider = &api.Azure{}
	case "hyper-v":
		provider = &hyperv.Provider{}
	case "upcloud":
		provider = upcloud.NewProvider()
	default:
		return provider, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := provider.Initialize(config)
	return provider, err
}

func getProviderAndContext(c *api.Config, providerName string) (api.Provider, *api.Context, error) {
	p, err := getCloudProvider(providerName, &c.CloudConfig)
	if err != nil {
		return nil, nil, err
	}

	ctx := api.NewContext(c)

	return p, ctx, nil
}

func initDefaultRunConfigs(c *api.Config, ports []string) {
	if c.RunConfig.Memory == "" {
		c.RunConfig.Memory = "2G"
	}
	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)
}

func fixupConfigImages(c *api.Config, version string) {
	if c.NightlyBuild {
		version = "nightly"
		c.Kernel = path.Join(api.GetOpsHome(), version, "kernel.img")
	}

	if c.Boot == "" {
		c.Boot = path.Join(api.GetOpsHome(), version, "boot.img")
	}

	if c.Kernel == "" {
		c.Kernel = path.Join(api.GetOpsHome(), version, "kernel.img")
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

// validateNetworkPorts verifies ports strings have right format
// Strings must have only numbers, commas or hyphens. Commas and hypens must separate 2 numbers
func validateNetworkPorts(ports []string) error {
	for _, str := range ports {
		var hyphenUsed bool

		if str[0] == ',' || str[len(str)-1] == ',' {
			return errors.Errorf("\"%s\" commas must separate numbers", str)
		} else if str[0] == '-' || str[len(str)-1] == '-' {
			return errors.Errorf("\"%s\" hyphen must separate two numbers", str)
		}

		for i, ch := range str {
			if ch == ',' {
				if !unicode.IsDigit(rune(str[i-1])) || !unicode.IsDigit(rune(str[i+1])) {
					return errors.Errorf("\"%s\" commas must separate numbers", str)
				}
			} else if ch == '-' {
				if hyphenUsed {
					return errors.Errorf("\"%s\" may have only one hyphen", str)
				} else if !unicode.IsDigit(rune(str[i-1])) || !unicode.IsDigit(rune(str[i+1])) {
					return errors.Errorf("\"%s\" hyphen must separate two numbers", str)
				}
				hyphenUsed = true
			} else if !unicode.IsDigit(ch) {
				return errors.Errorf("\"%s\" must have only numbers, commas or one hyphen", str)
			}
		}

	}

	return nil
}

// prepareNetworkPorts validates ports and split ports strings separated by commas
func prepareNetworkPorts(ports []string) (portsPrepared []string, err error) {
	err = validateNetworkPorts(ports)
	if err != nil {
		return
	}

	for _, ports := range ports {
		portsPrepared = append(portsPrepared, strings.Split(ports, ",")...)
	}

	return
}

// isIPAddressValid checks whether IP address is valid
func isIPAddressValid(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}

	return true
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

// SubtractTimeNotation subtracts time notation timestamp from date passed by argument
func SubtractTimeNotation(date time.Time, notation string) (newDate time.Time, err error) {
	r := regexp.MustCompile(`(\d+)(d|w|m|y){1}`)
	groups := r.FindStringSubmatch(notation)
	if len(groups) == 0 {
		return newDate, errors.New("invalid time notation")
	}

	n, err := strconv.Atoi(groups[1])
	if err != nil {
		fmt.Println(err)
		return newDate, err
	}

	switch groups[2] {
	case "d":
		return date.AddDate(0, 0, n*-1), nil
	case "w":
		return date.AddDate(0, 0, n*7*-1), nil
	case "m":
		return date.AddDate(0, n*-1, 0), nil
	case "y":
		return date.AddDate(n*-1, 0, 0), nil
	default:
		return date, nil
	}
}
