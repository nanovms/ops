package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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
	cmdBoot := &cobra.Command{
		Use:   "boot",
		Short: "Boot environment",
		Run:   envBoot,
	}

	cmdShutdown := &cobra.Command{
		Use:   "shutdown",
		Short: "Shutdown environment",
		Run:   envShutdown,
	}

	cmdResize := &cobra.Command{
		Use:   "resize [+|-]size[G/M]",
		Long:  "G for Gigabytes and M for Megabytes",
		Short: "Resize environment",
		Args:  cobra.MinimumNArgs(1),
		Run:   envResize,
	}
	cmdResize.PersistentFlags().BoolP("decrease", "d", false, "decrease disk size, default is to increase")

	cmdDependencies := &cobra.Command{
		Use:   "deps package1 package2 ... [flags]",
		Short: "Install/Uninstall dependencies",
		Args:  cobra.MinimumNArgs(1),
		Run:   envDependencies,
	}
	cmdDependencies.PersistentFlags().BoolP("uninstall", "", false, "Perform uninstallation, default is install")
	cmdDependencies.PersistentFlags().BoolP("npm", "", false, "Handle npm package")

	cmdBuild := &cobra.Command{
		Use:   "build executable_path [flags]",
		Short: "Build executable",
		Args:  cobra.MinimumNArgs(1),
		Run:   envBuild,
	}
	cmdBuild.PersistentFlags().StringP("type", "t", "", "Source type. Valid values is js")
	cmdBuild.PersistentFlags().BoolP("create-image", "", false, "Creates unikernel image using result executable")

	cmdRun := &cobra.Command{
		Use:   "run filename [flags]",
		Short: "Run source file",
		Args:  cobra.MinimumNArgs(1),
		Run:   envRun,
	}
	cmdRun.PersistentFlags().StringP("type", "t", "", "Source type. Valid values is js")

	cmd := &cobra.Command{
		Use:   "env command [flags]",
		Short: "Provides crossbuild environment",
		Args:  cobra.OnlyValidArgs,
		ValidArgs: []string{
			cmdBoot.Use,
			cmdShutdown.Use,
			cmdDependencies.Use,
			cmdBuild.Use,
			cmdResize.Use,
			cmdRun.Use,
		},
	}
	cmd.AddCommand(
		cmdBoot,
		cmdShutdown,
		cmdDependencies,
		cmdBuild,
		cmdResize,
		cmdRun,
	)
	return cmd
}

// Action for run command.
func envRun(cmd *cobra.Command, args []string) {
	env := loadEnvironment("")
	if err := env.Sync(); err != nil {
		log.Fail("failed to sync working directory", err)
	}

	filename := args[0]
	fileType := strings.TrimPrefix(filepath.Ext(filename), ".")

	sourceType, err := cmd.Flags().GetString("type")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}
	if sourceType != "" {
		fileType = sourceType
	}

	switch fileType {
	case "js":
		if err := env.RunUsingNexe(filename); err != nil {
			log.Fail("failed to run file", err)
		}
	default:
		exitWithError("unsupported source type, please use flags to force specific type")
	}
}

// Action for build command.
func envBuild(cmd *cobra.Command, args []string) {
	env := loadEnvironment("")
	if err := env.Sync(); err != nil {
		log.Fail("failed to sync working directory", err)
	}

	filename := args[0]
	ext := filepath.Ext(filename)
	fileType := strings.TrimPrefix(filepath.Ext(filename), ".")

	sourceType, err := cmd.Flags().GetString("type")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}
	if sourceType != "" {
		fileType = sourceType
	}

	switch fileType {
	case "js":
		if err := env.BuildUsingNexe(filename); err != nil {
			log.Fail("failed to build executable", err)
		}
	default:
		exitWithError("unsupported source type, please use flags to force specific type")
	}

	shouldBuildImg, err := cmd.Flags().GetBool("create-image")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}
	if shouldBuildImg {
		if err := env.OpsBuildImage(strings.TrimSuffix(filename, ext)); err != nil {
			log.Fail("failed to build image", err)
		}
	}
}

// Action for deps command.
func envDependencies(cmd *cobra.Command, args []string) {
	env := loadEnvironment("")
	if env.PID == 0 {
		fmt.Printf("environment '%s' is not running\n", env.Name)
		return
	}

	shouldUninstall, err := cmd.Flags().GetBool("uninstall")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	op := "install"
	if shouldUninstall {
		op = "uninstall"
	}
	errorMsg := fmt.Sprintf("failed to %s one or more dependency", op)

	npmRequested, err := cmd.Flags().GetBool("npm")
	if err != nil {
		exitForCmd(cmd, err.Error())
	}
	if npmRequested {
		for _, name := range args {
			if shouldUninstall {
				if err := env.UninstallNPMPackage(name); err != nil {
					log.Fail(errorMsg, err)
				}
			} else {
				if err := env.InstallNPMPackage(name); err != nil {
					log.Fail(errorMsg, err)
				}
			}
		}
		return
	}

	for _, name := range args {
		if shouldUninstall {
			if err := env.UninstallPackage(name); err != nil {
				log.Fail(errorMsg, err)
			}
		} else {
			if err := env.InstallPackage(name); err != nil {
				log.Fail(errorMsg, err)
			}
		}
	}
}

// Action for boot command.
func envBoot(cmd *cobra.Command, args []string) {
	var (
		env      = loadEnvironment("")
		progress = &util.ProgressSpinner{}
	)

	if env.PID > 0 {
		fmt.Printf("environment '%s' is running\n", env.Name)
		return
	}

	if err := progress.Do(func() error {
		return env.Boot()
	}, "booting environment: ", env.Name); err != nil {
		log.Fail(MsgFailedToLoadEnvironment, err)
	}
}

// Action for shutdown command.
func envShutdown(cmd *cobra.Command, args []string) {
	var (
		env      = loadEnvironment("")
		progress = &util.ProgressSpinner{}
	)

	if env.PID == 0 {
		fmt.Printf("environment '%s' is not running\n", env.Name)
		return
	}
	if err := progress.Do(func() error {
		return env.Shutdown()
	}, "shutting down environment"); err != nil {
		log.Fail("failed to shutdown environment", err)
	}
}

func envResize(cmd *cobra.Command, args []string) {
	var (
		env        = loadEnvironment("")
		progress   = &util.ProgressSpinner{}
		errMessage = "failed to resize environment"
	)

	if env.PID > 0 {
		if err := progress.Do(func() error {
			return env.Shutdown()
		}, "shutdown environment"); err != nil {
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

// Loads environment identified by given id.
func loadEnvironment(id string) *crossbuild.Environment {
	errorMsg := "failed to load requested environment"
	if id == "" {
		envDefault, err := crossbuild.DefaultEnvironment()
		if err != nil {
			log.Fail(errorMsg, err)
		}
		id = envDefault.ID
	}

	env, err := crossbuild.LoadEnvironment(id)
	if err != nil {
		log.Fail(errorMsg, err)
	}
	return env
}
