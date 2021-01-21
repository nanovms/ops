package cmd

import (
	"github.com/spf13/cobra"
)

// GetRootCommand provides set all commands for Ops
func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{Use: "ops"}

	// persist flags transversal to every command
	PersistGlobalCommandFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(BuildCommand())
	rootCmd.AddCommand(ImageCommands())
	rootCmd.AddCommand(InstanceCommands())
	rootCmd.AddCommand(LoadCommand())
	rootCmd.AddCommand(ProfileCommand())
	rootCmd.AddCommand(PackageCommands())
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(UpdateCommand())
	rootCmd.AddCommand(VersionCommand())
	rootCmd.AddCommand(VolumeCommands())

	return rootCmd
}
