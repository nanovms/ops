package cmd

import (
	"github.com/spf13/cobra"
)

// GetRootCommand provides set all commands for Ops
func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{Use: "ops"}
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(NetCommands())
	rootCmd.AddCommand(BuildCommand())
	rootCmd.AddCommand(ManifestCommand())
	rootCmd.AddCommand(VersionCommand())
	rootCmd.AddCommand(ProfileCommand())
	rootCmd.AddCommand(UpdateCommand())
	rootCmd.AddCommand(PackageCommands())
	rootCmd.AddCommand(LoadCommand())
	rootCmd.AddCommand(InstanceCommands())
	rootCmd.AddCommand(ImageCommands())
	rootCmd.AddCommand(StartCommand())
	return rootCmd
}
