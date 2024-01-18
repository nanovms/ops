package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// GetRootCommand provides set all commands for Ops
func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "ops",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := checkInvalidFlags(cmd, args); err != nil {
				return err
			}

			config := &types.Config{}

			configFlag, _ := cmd.Flags().GetString("config")
			configFlag = strings.TrimSpace(configFlag)

			if configFlag != "" {
				if err := unWarpConfig(configFlag, config); err != nil {
					return err
				}
			}

			globalFlags := NewGlobalCommandFlags(cmd.Flags())
			if err := globalFlags.MergeToConfig(config); err != nil {
				return err
			}

			log.InitDefault(os.Stdout, config)
			return nil
		},
	}

	// persist flags transversal to every command
	PersistGlobalCommandFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(BuildCommand())
	rootCmd.AddCommand(EnvCommand())
	rootCmd.AddCommand(ImageCommands())
	rootCmd.AddCommand(InstanceCommands())
	rootCmd.AddCommand(NetworkCommands())
	rootCmd.AddCommand(ProfileCommand())
	rootCmd.AddCommand(PackageCommands())
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(ComposeCommands())

	rootCmd.AddCommand(UpdateCommand())
	rootCmd.AddCommand(VersionCommand())
	rootCmd.AddCommand(VolumeCommands())
	rootCmd.AddCommand(DeployCommand())

	return rootCmd
}

func checkInvalidFlags(cmd *cobra.Command, args []string) error {
	length := len(args)
	requiresArg := strings.Contains(cmd.Use, "[")
	if length > 0 && requiresArg {
		if length == 1 {
			args = []string{}
		} else {
			args = args[1:]
		}
		length--
	}

	if length > 1 && requiresArg {
		message := "invalid argument%s or flag%s provided, use --help for usage information."
		plural := ""
		if length == 1 {
			plural = "s"
		}

		message = fmt.Sprintf(message, plural, plural)

		knownFlag, matched := checkKnownFlag(cmd, args)
		if matched {
			message = fmt.Sprintf("%s Probably what you mean was '-%s' instead of '%s'?", message, knownFlag, knownFlag)
		}
		return errors.New(message)

	}
	return nil
}

func checkKnownFlag(cmd *cobra.Command, args []string) (string, bool) {
	flags := make([]string, 0)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, f.Name, f.Shorthand)
	})

	for _, a := range args {
		for _, fl := range flags {
			if fl == a {
				return a, true
			}
		}
	}
	return "", false
}
