package cmd

import (
	"fmt"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Ops version: %s\n", api.Version)
	fmt.Printf("Nanos version: %s\n", api.LocalReleaseVersion)
}

// VersionCommand provides version command
func VersionCommand() *cobra.Command {
	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Version",
		Run:   printVersion,
	}
	return cmdVersion
}
