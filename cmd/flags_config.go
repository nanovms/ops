package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/onprem"
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
)

// ConfigCommandFlags handles config file path flag and build configuration from the file
type ConfigCommandFlags struct {
	Config string
}

// MergeToConfig reads a json configuration file
func (flags *ConfigCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if flags.Config != "" {

		if c == nil {
			c = &types.Config{}
		}

		unWarpConfig(flags.Config, c)

		return
	} else if c == nil {
		*c = *lepton.NewConfig()
	}

	return
}

// unWarpConfig parses lepton config file from file
func unWarpConfig(file string, c *types.Config) (err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return fmt.Errorf("error config: %v", err)
	}

	c.VolumesDir = lepton.LocalVolumeDir
	if c.Mounts != nil {
		err = onprem.AddMountsFromConfig(c)
	}

	if c.RunConfig.IPAddress != "" && !isIPAddressValid(c.RunConfig.IPAddress) {
		c.RunConfig.IPAddress = ""
	}

	if c.RunConfig.Gateway != "" && !isIPAddressValid(c.RunConfig.Gateway) {
		c.RunConfig.Gateway = ""
	}

	if c.RunConfig.NetMask != "" && !isIPAddressValid(c.RunConfig.NetMask) {
		c.RunConfig.NetMask = ""
	}

	return
}

// NewConfigCommandFlags returns an instance of ConfigCommandFlags
func NewConfigCommandFlags(cmdFlags *pflag.FlagSet) (flags *ConfigCommandFlags) {
	var err error
	flags = &ConfigCommandFlags{}

	flags.Config, err = cmdFlags.GetString("config")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Config = strings.TrimSpace(flags.Config)

	return
}

// PersistConfigCommandFlags append a command the required flags to run an image
func PersistConfigCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringP("config", "c", "", "ops config file")
}
