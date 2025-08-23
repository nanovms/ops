package cmd

import (
	"fmt"
	"os"

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

	if len(args) != 2 {
		fmt.Println("both image and schedule are required")
		fmt.Println("for example myimg rate(1 minutes)")
		os.Exit(1)
	}

	c.CloudConfig.ImageName = args[0]
	schedule := args[1]

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, providerFlags, pkgFlags)

	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p := aws.NewProvider()
	ctx := api.NewContext(c)
	err = p.Initialize(&c.CloudConfig)

	err = p.CreateCron(ctx, c.CloudConfig.ImageName, schedule)
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
		Run:   cronDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

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

	if len(args) != 1 {
		fmt.Println("cron name needs to be specified")
		os.Exit(1)
	}

	cron := args[0]

	errMsg := p.DeleteCron(ctx, cron)
	if errMsg != nil {
		errMsg = fmt.Errorf("failed deleting %s: %v", "", errMsg)
	}

}

func cronEnableCommand() *cobra.Command {
	var cmdCronEnable = &cobra.Command{
		Use:   "enable <cron_name>",
		Short: "enable cron from provider",
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
		Use:   "disable <crone_name>",
		Short: "disable cron from provider",
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
