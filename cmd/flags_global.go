package cmd

import (
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// GlobalCommandFlags are flags accepted by every command
type GlobalCommandFlags struct {
	ShowWarnings bool
	ShowErrors   bool
	ShowDebug    bool
}

// MergeToConfig append command flags that are used transversally for all commands to configuration
func (flags *GlobalCommandFlags) MergeToConfig(config *lepton.Config) (err error) {
	config.RunConfig.ShowWarnings = flags.ShowWarnings
	config.RunConfig.ShowErrors = flags.ShowErrors
	config.RunConfig.ShowDebug = flags.ShowDebug

	return
}

// NewGlobalCommandFlags returns an instance of GlobalCommandFlags
func NewGlobalCommandFlags(cmdFlags *pflag.FlagSet) (flags *GlobalCommandFlags) {
	flags = &GlobalCommandFlags{}

	flags.ShowWarnings, _ = cmdFlags.GetBool("show-warnings")
	flags.ShowErrors, _ = cmdFlags.GetBool("show-errors")
	flags.ShowDebug, _ = cmdFlags.GetBool("show-debug")

	return flags
}

// PersistGlobalCommandFlags append the global flags to a command
func PersistGlobalCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.Bool("show-warnings", false, "display warning messages")
	cmdFlags.Bool("show-errors", false, "display error messages")
	cmdFlags.Bool("show-debug", false, "display debug messages")
}
