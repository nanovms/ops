package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/ttacon/chalk"
)

// NightlyCommandFlags is used to change binaries paths used on ops operations
type NightlyCommandFlags struct {
	Nightly bool
}

// MergeToConfig append command flags that are used transversally for all commands to configuration
func (flags *NightlyCommandFlags) MergeToConfig(config *lepton.Config) (err error) {
	config.NightlyBuild = flags.Nightly

	setNanosBaseImage(config)

	return
}

// NewNightlyCommandFlags returns an instance of NightlyCommandFlags
func NewNightlyCommandFlags(cmdFlags *pflag.FlagSet) (flags *NightlyCommandFlags) {
	var err error
	flags = &NightlyCommandFlags{}

	flags.Nightly, err = cmdFlags.GetBool("nightly")
	if err != nil {
		exitWithError(err.Error())
	}

	return
}

func setNanosBaseImage(c *api.Config) {
	var err error
	var currversion string

	if c.NightlyBuild {
		currversion, err = downloadNightlyImages(c)
	} else {
		currversion, err = downloadReleaseImages()
	}

	panicOnError(err)
	updateNanosToolsPaths(c, currversion)
}

func updateNanosToolsPaths(c *api.Config, version string) {
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

	if _, err := os.Stat(c.Kernel); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Kernel, err)
		os.Exit(1)
	}

	if _, err := os.Stat(c.Boot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", c.Boot, err)
		os.Exit(1)
	}

	if c.NameServer == "" {
		// google dns server
		c.NameServer = "8.8.8.8"
	}
}

func downloadNightlyImages(c *api.Config) (string, error) {
	var err error
	err = api.DownloadNightlyImages(c)
	return "nightly", err
}

func parseVersion(s string, width int) int64 {
	strList := strings.Split(s, ".")
	format := fmt.Sprintf("%%s%%0%ds", width)
	v := ""
	for _, value := range strList {
		v = fmt.Sprintf(format, v, value)
	}
	var result int64
	var err error
	if result, err = strconv.ParseInt(v, 10, 64); err != nil {
		panic(err)
	}
	return result
}

func downloadReleaseImages() (string, error) {
	var err error

	// if it's first run or we have an update
	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	if local == "0.0" {
		err = api.DownloadReleaseImages(remote)
		if err != nil {
			return "", err
		}
		return remote, nil

	}

	if parseVersion(local, 4) != parseVersion(remote, 4) {
		fmt.Println(chalk.Red, "You are running an older version of Ops.", chalk.Reset)
		fmt.Println(chalk.Red, "Update: Run", chalk.Reset, chalk.Bold.TextStyle("`ops update`"))
	}

	return local, nil
}

// PersistNightlyCommandFlags append nightly flag to a command
func PersistNightlyCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.BoolP("nightly", "n", false, "nightly build")
}
