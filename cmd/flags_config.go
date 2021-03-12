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

		*c = *unWarpConfig(flags.Config)

		return
	}

	if c == nil {
		*c = *lepton.NewConfig()
	}

	return
}

// unWarpConfig parses lepton config file from file
func unWarpConfig(file string) *types.Config {
	c := types.Config{}
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

	c.VolumesDir = lepton.LocalVolumeDir
	if c.Mounts != nil {
		err = onprem.AddMountsFromConfig(&c)
	}

	return &c
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
