package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

	"github.com/spf13/cobra"
)

// InstanceCommands provides instance related commands
func InstanceCommands() *cobra.Command {

	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"create", "list", "delete", "stop", "start", "logs"},
		Args:      cobra.OnlyValidArgs,
	}

	PersistProviderCommandFlags(cmdInstance.PersistentFlags())
	PersistConfigCommandFlags(cmdInstance.PersistentFlags())

	cmdInstance.AddCommand(instanceCreateCommand())
	cmdInstance.AddCommand(instanceListCommand())
	cmdInstance.AddCommand(instanceDeleteCommand())
	cmdInstance.AddCommand(instanceStopCommand())
	cmdInstance.AddCommand(instanceStartCommand())
	cmdInstance.AddCommand(instanceLogsCommand())

	return cmdInstance
}

func instanceCreateCommand() *cobra.Command {

	var cmdInstanceCreate = &cobra.Command{
		Use:   "create <instance_name>",
		Short: "create nanos instance",
		Run:   instanceCreateCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	persistentFlags := cmdInstanceCreate.PersistentFlags()
	PersistCreateInstanceFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	cmdInstanceCreate.PersistentFlags().StringP("instance-name", "i", "", "instance name (overrides default instance name format)")
	cmdInstanceCreate.PersistentFlags().StringP("instance-group", "", "", "instance group")

	return cmdInstanceCreate
}

func instanceCreateCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	createInstanceFlags := NewCreateInstanceCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)

	c := lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, providerFlags, createInstanceFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	c.CloudConfig.ImageName = args[0]

	instanceName, _ := cmd.Flags().GetString("instance-name")

	instanceGroup, _ := cmd.Flags().GetString("instance-group")

	if instanceName == "" {
		c.RunConfig.InstanceName = fmt.Sprintf("%v-%v",
			strings.Split(filepath.Base(c.CloudConfig.ImageName), ".")[0],
			strconv.FormatInt(time.Now().Unix(), 10),
		)
	} else {
		c.RunConfig.InstanceName = instanceName
	}

	if instanceGroup != "" {
		c.RunConfig.InstanceGroup = instanceGroup
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	c.RunConfig.Kernel = c.Kernel

	err = p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}


	fmt.Printf("%s instance '%s' created...\n", c.CloudConfig.Platform, c.RunConfig.InstanceName)
}

func instanceListCommand() *cobra.Command {
	var cmdInstanceList = &cobra.Command{
		Use:   "list",
		Short: "list instance on provider",
		Run:   instanceListCommandHandler,
	}
	return cmdInstanceList
}

func instanceListCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.ListInstances(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceDeleteCommand() *cobra.Command {
	var cmdInstanceDelete = &cobra.Command{
		Use:   "delete <instance_name>",
		Short: "delete instance on provider",
		Run:   instanceDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	PersistDeleteInstanceFlags(cmdInstanceDelete.PersistentFlags())
	return cmdInstanceDelete
}

func instanceDeleteCommandHandler(cmd *cobra.Command, args []string) {

	c, err := getInstanceDeleteCommandConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.DeleteInstance(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceStartCommand() *cobra.Command {
	var cmdInstanceStart = &cobra.Command{
		Use:   "start <instance_name>",
		Short: "start instance on provider",
		Run:   instanceStartCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdInstanceStart
}

func instanceStartCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.StartInstance(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceStopCommand() *cobra.Command {
	var cmdInstanceStop = &cobra.Command{
		Use:   "stop <instance_name>",
		Short: "stop instance on provider",
		Run:   instanceStopCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdInstanceStop
}

func instanceStopCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.StopInstance(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceLogsCommand() *cobra.Command {
	var watch bool
	var cmdLogsCommand = &cobra.Command{
		Use:   "logs <instance_name>",
		Short: "Show logs from console for an instance",
		Run:   instanceLogsCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdLogsCommand.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "watch logs")
	return cmdLogsCommand
}

func instanceLogsCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	watch, err := strconv.ParseBool(cmd.Flag("watch").Value.String())
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.PrintInstanceLogs(ctx, args[0], watch)
	if err != nil {
		exitWithError(err.Error())
	}
}

func getInstanceCommandDefaultConfig(cmd *cobra.Command) (c *types.Config, err error) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c = lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err = mergeContainer.Merge(c)

	return
}

func getInstanceDeleteCommandConfig(cmd *cobra.Command) (c *types.Config, err error) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	deleteFlags := NewDeleteInstanceCommandFlags(flags)

	c = lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags, deleteFlags)
	err = mergeContainer.Merge(c)

	return
}
