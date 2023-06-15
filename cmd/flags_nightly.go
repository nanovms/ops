package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
	"github.com/ttacon/chalk"
)

// NightlyCommandFlags is used to change configuration to use nightly build tools paths
type NightlyCommandFlags struct {
	Nightly bool
	Arch    string
}

// MergeToConfig downloads nightly build and change configuration nanos tools paths
func (flags *NightlyCommandFlags) MergeToConfig(config *types.Config) (err error) {
	if flags.Nightly {
		config.NightlyBuild = flags.Nightly
	}

	if flags.Arch != "" {
		api.AltGOARCH = flags.Arch
		// this is dumb, FIXME
		api.NightlyLocalFolderm = api.NightlyLocalFolder()
		api.NightlyReleaseURLm = api.NightlyReleaseURL()
	}

	if config.NightlyBuild {
		var version string
		version, err = downloadNightlyImages(config)
		if err != nil {
			return
		}
		updateNanosToolsPaths(config, version)
	}

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

	flags.Arch, err = cmdFlags.GetString("arch")
	if err != nil {
		exitWithError(err.Error())
	}

	return
}

// PersistNightlyCommandFlags append nightly flag to a command
func PersistNightlyCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.BoolP("nightly", "n", false, "nightly build")
	cmdFlags.StringP("arch", "", "", "set different architecture")
}

func getKernelVersion(version string) string {
	return path.Join(api.GetOpsHome(), version, "kernel.img")
}

func setKernelVersion(version string) string {
	if api.AltGOARCH != "" {
		if runtime.GOARCH == "amd64" {
			return version + "-arm"
		}
		return version
	}
	if runtime.GOARCH == "arm64" {
		return version + "-arm"
	}
	return version
}

func updateNanosToolsPaths(c *types.Config, version string) {
	if c.NightlyBuild {
		version = "nightly"
	}

	version = setKernelVersion(version)

	if c.Boot == "" {
		bootPath := path.Join(api.GetOpsHome(), version, "boot.img")
		if _, err := os.Stat(bootPath); err == nil {
			c.Boot = bootPath
		}
	}

	c.UefiBoot = api.GetUefiBoot(version)

	if c.Kernel == "" {
		c.Kernel = getKernelVersion(version)
	}

	if _, err := os.Stat(c.Kernel); os.IsNotExist(err) {
		log.Fatalf("error: %v: %v\n", c.Kernel, err)
	}

	if c.Boot != "" {
		if _, err := os.Stat(c.Boot); os.IsNotExist(err) {
			log.Fatalf("error: boot file '%v' not found\n", c.Boot)
		}
	}

	if len(c.NameServers) == 0 {
		// google dns server
		c.NameServers = []string{"8.8.8.8"}
	}
}

func downloadNightlyImages(c *types.Config) (string, error) {
	var err error
	err = api.DownloadNightlyImages(c)

	if runtime.GOARCH == "arm64" || api.AltGOARCH == "arm64" {
		return "nightly-arm", err
	}
	return "nightly", err

}

func getCurrentVersion() (string, error) {
	var err error

	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	if local == "0.0" {
		err = api.DownloadReleaseImages(remote, "")
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
		fmt.Printf("Failed to parse version %s, error is: %s", v, err)
		os.Exit(1)
	}
	return result
}
