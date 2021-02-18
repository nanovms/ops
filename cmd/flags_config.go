package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/nanovms/ops/config"
	"github.com/spf13/pflag"
)

// ConfigCommandFlags handles config file path flag and build configuration from the file
type ConfigCommandFlags struct {
	Config string
}

// MergeToConfig reads a json configuration file
func (flags *ConfigCommandFlags) MergeToConfig(c *config.Config) (err error) {
	if flags.Config != "" {
		var data []byte

		if c == nil {
			*c = *config.NewConfig()
		}

		data, err = ioutil.ReadFile(flags.Config)
		if err != nil {
			err = fmt.Errorf("error reading config: %v", err)
			return
		}

		err = json.Unmarshal(data, &c)
		if err != nil {
			err = fmt.Errorf("error config: %v", err)
			return
		}
		return
	}

	if c == nil {
		*c = *unWarpDefaultConfig()
	}

	return

}

// unWarpConfig parses lepton config file from file
func unWarpConfig(file string) *config.Config {
	var c config.Config
	if file != "" {
		c = *config.NewConfig()
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
func unWarpDefaultConfig() *config.Config {
	c := *config.NewConfig()
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
