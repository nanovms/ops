package cmd

import (
	"os"
	"strings"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
)

// GetRootCommand provides set all commands for Ops
func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "ops",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var config *types.Config

			configFlag, _ := cmd.Flags().GetString("config")
			configFlag = strings.TrimSpace(configFlag)

			if configFlag != "" {
				config := &types.Config{}
				if err := unWarpConfig(configFlag, config); err != nil {
					return err
				}

				globalFlags := NewGlobalCommandFlags(cmd.Flags())
				if err := globalFlags.MergeToConfig(config); err != nil {
					return err
				}
			}

			log.InitDefault(os.Stdout, config)
			return nil
		},
	}

	// persist flags transversal to every command
	PersistGlobalCommandFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(BuildCommand())
	rootCmd.AddCommand(ImageCommands())
	rootCmd.AddCommand(InstanceCommands())
	rootCmd.AddCommand(ProfileCommand())
	rootCmd.AddCommand(PackageCommands())
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(UpdateCommand())
	rootCmd.AddCommand(VersionCommand())
	rootCmd.AddCommand(VolumeCommands())
	rootCmd.AddCommand(DeployCommand())

	return rootCmd
}
