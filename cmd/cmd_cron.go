package cmd

import (
	"fmt"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/aws"
	"github.com/spf13/cobra"
)

// CronCommands provides cron related commands
func CronCommands() *cobra.Command {
	var cmdCron = &cobra.Command{
		Use:       "cron",
		Short:     "manage cronjobs",
		ValidArgs: []string{"create", "list", "delete", "enable", "disable"},
		Args:      cobra.OnlyValidArgs,
	}

	PersistConfigCommandFlags(cmdCron.PersistentFlags())
	PersistProviderCommandFlags(cmdCron.PersistentFlags())

	cmdCron.AddCommand(cronCreateCommand())
	cmdCron.AddCommand(cronListCommand())
	cmdCron.AddCommand(cronDeleteCommand())
	cmdCron.AddCommand(cronEnableCommand())
	cmdCron.AddCommand(cronDisableCommand())

	return cmdCron
}

func cronCreateCommand() *cobra.Command {
	// should take cron as well

	var cmdCronCreate = &cobra.Command{
		Use:   "create <cron_name>",
		Short: "create cron from image",
		Run:   cronCreateCommandHandler,
	}

	persistentFlags := cmdCronCreate.PersistentFlags()

	PersistBuildImageCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)
	PersistPkgCommandFlags(persistentFlags)

	return cmdCronCreate
}

func cronCreateCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	c.CloudConfig.ImageName = args[0]
	// schedule = args[1]
	schedule := "rate(1 minutes)"

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, providerFlags, pkgFlags)

	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p := aws.NewProvider()
	ctx := api.NewContext(c)
	err = p.Initialize(&c.CloudConfig)

	err = p.CreateCron(ctx, schedule)
	if err != nil {
		exitWithError(err.Error())
	}

}

func cronListCommand() *cobra.Command {
	var cmdCronList = &cobra.Command{
		Use:   "list",
		Short: "list crons from provider",
		Run:   cronListCommandHandler,
	}
	return cmdCronList
}

func cronListCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p := aws.NewProvider()
	ctx := api.NewContext(c)

	err = p.ListCrons(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func cronDeleteCommand() *cobra.Command {
	var cmdCronDelete = &cobra.Command{
		Use:   "delete <image_name>",
		Short: "delete images from provider",
		Run:   imageDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	cmdCronDelete.PersistentFlags().StringP("lru", "", "", "clean least recently used images with a time notation. Use \"1w\" notation to delete images older than one week. Other notation examples are 300d, 3w, 1m and 2y.")
	cmdCronDelete.PersistentFlags().BoolP("assume-yes", "", false, "clean images without waiting for confirmation")
	cmdCronDelete.PersistentFlags().BoolP("force", "", false, "force even if image is being used by instance")

	return cmdCronDelete
}

func cronDeleteCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p := aws.NewProvider()
	ctx := api.NewContext(c)

	errMsg := p.DeleteCron(ctx, "itesting")
	if errMsg != nil {
		errMsg = fmt.Errorf("failed deleting %s: %v", "", errMsg)
	}

}

func cronEnableCommand() *cobra.Command {
	var cmdCronEnable = &cobra.Command{
		Use:   "enable <image_name>",
		Short: "enable image from provider",
		Run:   cronEnableCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdCronEnable
}

func cronEnableCommandHandler(cmd *cobra.Command, args []string) {
	c := api.NewConfig()

	p := aws.NewProvider()
	ctx := api.NewContext(c)

	cronName := args[0]

	err := p.EnableCron(ctx, cronName)
	if err != nil {
		exitWithError(err.Error())
	}

}

func cronDisableCommand() *cobra.Command {
	var cmdCronDisable = &cobra.Command{
		Use:   "disable <image_name>",
		Short: "disable image from provider",
		Run:   cronDisableCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdCronDisable
}

func cronDisableCommandHandler(cmd *cobra.Command, args []string) {
	c := api.NewConfig()

	p := aws.NewProvider()
	ctx := api.NewContext(c)

	cronName := args[0]

	err := p.DisableCron(ctx, cronName)
	if err != nil {
		exitWithError(err.Error())
	}

}
