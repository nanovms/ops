package cmd

import (
	"fmt"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println(api.Version)
}
func VersionCommand() *cobra.Command {
	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Version",
		Run:   printVersion,
	}
	return cmdVersion
}
