package cmd

import (
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// AppendGlobalCmdFlagsToConfig append command flags that are used transversally for all commands to configuration
func AppendGlobalCmdFlagsToConfig(cmdFlags *pflag.FlagSet, config *lepton.Config) {
	config.RunConfig.ShowWarnings, _ = cmdFlags.GetBool("show-warnings")
	config.RunConfig.ShowErrors, _ = cmdFlags.GetBool("show-errors")
}
