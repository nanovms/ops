package cmd

import (
	"fmt"

	"github.com/nanovms/ops/crossbuild"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/util"
	"github.com/spf13/cobra"
)

var (
	// MsgFailedToLoadEnvironment printed if environment loading failed.
	MsgFailedToLoadEnvironment = "Failed to load environment"
)

// EnvCommand provides cross-build features.
func EnvCommand() *cobra.Command {
	cmdLoad := &cobra.Command{
		Use:   "load",
		Short: "Load existing environment, or creates one if not exists",
		Run:   envLoad,
	}

	cmdShutdown := &cobra.Command{
		Use:   "shutdown",
		Short: "Shutdown environment",
		Run:   envShutdown,
	}

	cmdRemove := &cobra.Command{
		Use:   "remove",
		Short: "Removes environment",
		Run:   envRemove,
	}

	cmdResize := &cobra.Command{
		Use:   "resize",
		Short: "Resize environment",
		Args:  cobra.MinimumNArgs(1),
		Run:   envResize,
	}
	cmdResize.PersistentFlags().BoolP("decrease", "d", false, "decrease disk size, default is to increase")

	cmdRun := &cobra.Command{
		Use:   "run",
		Short: "Execute run command",
		Run:   envRun,
	}

	cmdInstallDeps := &cobra.Command{
		Use:   "deps-install",
		Short: "Install dependencies",
		Run:   envInstallDependencies,
	}

	cmd := &cobra.Command{
		Use:   "env",
		Short: "Provides crossbuild environment",
		Args:  cobra.OnlyValidArgs,
		ValidArgs: []string{
			cmdLoad.Use,
			cmdShutdown.Use,
			cmdRemove.Use,
			cmdResize.Use,
			cmdRun.Use,
			cmdInstallDeps.Use,
		},
	}
	cmd.AddCommand(
		cmdLoad,
		cmdShutdown,
		cmdRemove,
		cmdResize,
		cmdRun,
		cmdInstallDeps,
	)
	return cmd
}

// Action for load command.
func envLoad(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(true)
	defer progress.Close()

	if err := progress.Do(func() error {
		return env.Sync()
	}, "Sync directory"); err != nil {
		log.Fail(MsgFailedToLoadEnvironment, err)
	}
}

// Action for shutdown command.
func envShutdown(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(false)
	defer progress.Close()

	if err := progress.Do(func() error {
		return env.Shutdown()
	}, "Shutdown environment"); err != nil {
		log.Fail("Failed to shutdown environment", err)
	}
}

// Action for remove command.
func envRemove(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(true)
	defer progress.Close()

	errMessage := "Failed to remove environment"
	if err := progress.Do(func() error {
		return env.Remove()
	}, "Remove resources"); err != nil {
		log.Fail(errMessage, err)
	}

	if err := progress.Do(func() error {
		return env.Shutdown()
	}, "Shutdown environment"); err != nil {
		log.Fail(errMessage, err)
	}
}

// Action for sizeinc
func envResize(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(false)
	defer progress.Close()

	var (
		errMessage = "Failed to increase environment size"
	)

	if env.Running() {
		if err := progress.Do(func() error {
			return env.Shutdown()
		}, "Shutdown environment"); err != nil {
			log.Fail(errMessage, err)
		}
	}

	shouldDecrease, err := cmd.Flags().GetBool("decrease")
	if err != nil {
		exitWithError(err.Error())
	}

	op := "+"
	if shouldDecrease {
		op = "-"
	}
	if err := env.Resize(op + args[0]); err != nil {
		log.Fail(errMessage, err)
	}
}

// Action for run command
func envRun(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(true)
	defer progress.Close()

	if err := env.Run(); err != nil {
		log.Fail("Failed to execute run command", err)
	}
}

// Action for run command
func envInstallDependencies(cmd *cobra.Command, args []string) {
	env, progress := loadEnvironment(true)
	defer progress.Close()

	if err := env.InstallDependencies(); err != nil {
		log.Fail("Failed to install dependencies", err)
	}
}

func loadEnvironment(startVM bool) (*crossbuild.Environment, *util.ProgressSpinner) {
	progress := &util.ProgressSpinner{}

	env, err := crossbuild.LoadEnvironment()
	if err != nil {
		log.Fail(MsgFailedToLoadEnvironment, err)
	}

	if !env.Exists() {
		fmt.Println("Download environment: ", env.Name)
		if err := env.Download(); err != nil {
			log.Fail(MsgFailedToLoadEnvironment, err)
		}
	}

	if startVM && !env.Running() {
		if err := progress.Do(func() error {
			return env.Start()
		}, "Start environment: ", env.Name); err != nil {
			log.Fail(MsgFailedToLoadEnvironment, err)
		}
	}

	return env, progress
}
