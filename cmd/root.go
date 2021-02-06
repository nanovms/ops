package cmd

import (
	"github.com/spf13/cobra"
)

// GetRootCommand provides set all commands for Ops
func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{Use: "ops"}

	// persist flags transversal to every command
	rootCmd.PersistentFlags().Bool("show-warnings", false, "display warning messages")
	rootCmd.PersistentFlags().Bool("show-errors", false, "display error messages")
	rootCmd.PersistentFlags().Bool("show-debug", false, "display debug messages")

	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(NetCommands())
	rootCmd.AddCommand(BuildCommand())
	rootCmd.AddCommand(VersionCommand())
	rootCmd.AddCommand(ProfileCommand())
	rootCmd.AddCommand(UpdateCommand())
	rootCmd.AddCommand(PackageCommands())
	rootCmd.AddCommand(LoadCommand())
	rootCmd.AddCommand(InstanceCommands())
	rootCmd.AddCommand(ImageCommands())
	rootCmd.AddCommand(VolumeCommands())

	return rootCmd
}
