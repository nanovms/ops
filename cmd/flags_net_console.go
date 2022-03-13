package cmd

import (
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
)

// NetConsoleFlags for flags related to net console options
type NetConsoleFlags struct {
	NetConsolePort int
	NetConsoleIP   string
	Consoles       []string
}

// NewNetConsoleFlags returns an instance of NetConsoleFlags
func NewNetConsoleFlags(cmdFlags *pflag.FlagSet) (flags *NetConsoleFlags) {
	var err error
	flags = &NetConsoleFlags{}

	flags.NetConsolePort, err = cmdFlags.GetInt("netconsole-port")
	if err != nil {
		exitWithError(err.Error())
	}
	flags.NetConsoleIP, err = cmdFlags.GetString("netconsole-ip")
	if err != nil {
		exitWithError(err.Error())
	}

	flags.Consoles, err = cmdFlags.GetStringArray("consoles")
	if err != nil {
		exitWithError(err.Error())
	}
	return
}

// MergeToConfig sets passthrough data in the config from the flags
func (flags *NetConsoleFlags) MergeToConfig(config *types.Config) (err error) {
	if config.ManifestPassthrough == nil {
		config.ManifestPassthrough = make(map[string]interface{})
	}
	config.ManifestPassthrough["netconsole-port"] = flags.NetConsolePort
	config.ManifestPassthrough["netconsole-ip"] = flags.NetConsoleIP
	config.ManifestPassthrough["consoles"] = flags.Consoles
	return
}

// PersistNetConsoleFlags append flag to a command
func PersistNetConsoleFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.IntP("netconsole-port", "", 4444, "set net console port")
	cmdFlags.StringP("netconsole-ip", "", "10.0.2.2", "set net console ip")
	cmdFlags.StringArrayP("consoles", "", []string{}, "set different consoles to forward logs to")
}
