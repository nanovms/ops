package cmd

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/crossbuild"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/spf13/cobra"
)

// EnvCommand provides cross-build features.
func EnvCommand() *cobra.Command {
	cmdInstall := &cobra.Command{
		Use:   "install",
		Short: "Install environment",
		Run:   envInstall,
	}

	cmdBuild := &cobra.Command{
		Use:   "build <build_file> [flags]",
		Short: "Build application in environment",
		Args:  cobra.MinimumNArgs(1),
		Run:   envBuild,
	}
	flags := cmdBuild.PersistentFlags()
	flags.BoolP("create-image", "", false, "create Nanos image from environment")
	PersistConfigCommandFlags(flags)
	PersistNightlyCommandFlags(flags)
	PersistNanosVersionCommandFlags(flags)
	PersistBuildImageCommandFlags(flags)

	cmdCopy := &cobra.Command{
		Use:   "copy <local_dir> [flags]",
		Short: "Copy files from environment to local directory",
		Args:  cobra.MinimumNArgs(1),
		Run:   envCopy,
	}
	flags = cmdCopy.PersistentFlags()
	PersistConfigCommandFlags(flags)

	cmdUninstall := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall environment",
		Run:   envUninstall,
	}

	cmd := &cobra.Command{
		Use:   "env command [flags]",
		Short: "Cross-build environment commands",
		Args:  cobra.OnlyValidArgs,
		ValidArgs: []string{
			cmdInstall.Use,
			cmdBuild.Use,
			cmdCopy.Use,
			cmdUninstall.Use,
		},
	}
	cmd.AddCommand(
		cmdInstall,
		cmdBuild,
		cmdCopy,
		cmdUninstall,
	)
	return cmd
}

// Action for install command.
func envInstall(cmd *cobra.Command, args []string) {
	env := loadEnvironment(false)
	if env.IsInstalled() {
		exitWithError("Enviroment already installed")
	}
	if err := env.Install(); err != nil {
		exitWithError(fmt.Sprintf("Failed to install environment: %v", err))
	}
}

// Action for build command.
func envBuild(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	createImage, _ := flags.GetBool("create-image")
	c := api.NewConfig()
	configFlags := NewConfigCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	mergeContainer := NewMergeConfigContainer(configFlags, nightlyFlags, nanosVersionFlags, buildImageFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}
	env := loadEnvironment(true)
	if createImage {
		if c.Program == "" {
			exitForCmd(cmd, "When creating an image, program must be specified in config file")
		}
		if c.RunConfig.Imagename == "" {
			c.RunConfig.Imagename = c.Program
		}
	}
	tmpRoot := false
	if c.TargetRoot == "" {
		if createImage {
			c.TargetRoot, err = os.MkdirTemp("", "*")
			if err != nil {
				exitWithError(err.Error())
			}
			tmpRoot = true
		} else {
			exitForCmd(cmd, "When not creating an image, target root must be specified "+
				"(WARNING: any files in the target root folder will be overwritten)")
		}
	}
	if err = env.Boot(); err != nil {
		log.Errorf("Failed to start environment: %v", err)
		env = nil
		goto cleanup
	}
	if err = env.RunCommands(args[0]); err != nil {
		log.Errorf("Failed to build application: %v", err)
		goto cleanup
	}
	if err = env.GetFiles(c, c.TargetRoot); err != nil {
		log.Errorf("Failed to get application files: %v", err)
		goto cleanup
	}
	if createImage {
		c.LocalFilesParentDirectory = c.TargetRoot
		var p api.Provider
		var ctx *api.Context
		p, ctx, err = getProviderAndContext(c, "onprem")
		if err != nil {
			log.Errorf("Failed to get provider: %v", err)
			goto cleanup
		}
		_, err = p.BuildImage(ctx)
		if err != nil {
			log.Errorf("Failed to build image: %v", err)
		} else {
			fmt.Printf("On-prem image '%s' created...\n", c.RunConfig.Imagename)
		}
	}
cleanup:
	if env != nil {
		env.Shutdown()
	}
	if tmpRoot {
		os.RemoveAll(c.TargetRoot)
	}
	if err != nil {
		os.Exit(1)
	}
}

// Action for copy command.
func envCopy(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	c := api.NewConfig()
	configFlags := NewConfigCommandFlags(flags)
	mergeContainer := NewMergeConfigContainer(configFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}
	env := loadEnvironment(true)
	if err = env.Boot(); err != nil {
		log.Errorf("Failed to start environment: %v", err)
		os.Exit(1)
	}
	if err = env.GetFiles(c, args[0]); err != nil {
		log.Errorf("Failed to get application files: %v", err)
	}
	env.Shutdown()
	if err != nil {
		os.Exit(1)
	}
}

// Action for uninstall command.
func envUninstall(cmd *cobra.Command, args []string) {
	env := loadEnvironment(true)
	if err := env.Uninstall(); err != nil {
		exitWithError(fmt.Sprintf("Failed to uninstall environment: %v", err))
	}
}

// Loads environment, optionally checking that it is installed.
func loadEnvironment(checkInstall bool) *crossbuild.Environment {
	env, err := crossbuild.DefaultEnvironment()
	if err != nil {
		exitWithError(fmt.Sprintf("Failed to load environment: %v", err))
	}
	if checkInstall && !env.IsInstalled() {
		exitWithError("Enviroment not installed")
	}
	return env
}
