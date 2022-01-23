package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/onprem"
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
)

var (
	// ErrInvalidFileConfig is used when some error occurred on reading the configuration file. The message also provides instructions to search for how to set up configuration.
	ErrInvalidFileConfig = func(err error) error {
		n := 1
		for {
			_, file, line, _ := runtime.Caller(n)
			if file == "" {
				break
			}
			err = fmt.Errorf("%w %v: %v \n", err, file, line)
			n++
		}

		return fmt.Errorf("failed converting configuration file: %v\nSee more details at https://nanovms.gitbook.io/ops/configuration", err)
	}
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

		err = unWarpConfig(flags.Config, c)

		c.LocalFilesParentDirectory = path.Dir(flags.Config)

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
		log.Fatalf("error reading config: %v\n", err)
	}

	return ConvertJSONToConfig(data, c)
}

// ConvertJSONToConfig converts a byte array to an object of type configuration
func ConvertJSONToConfig(data []byte, c *types.Config) (err error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	err = dec.Decode(&c)
	if err != nil {
		return ErrInvalidFileConfig(err)
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
