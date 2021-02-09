package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nanovms/ops/lepton"
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
	cmdInstance.PersistentFlags().StringP("projectid", "g", os.Getenv("GOOGLE_CLOUD_PROJECT"), "project-id for GCP or set env GOOGLE_CLOUD_PROJECT")
	cmdInstance.PersistentFlags().StringP("zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for GCP or set env GOOGLE_CLOUD_ZONE")

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
	}

	cmdInstanceCreate.PersistentFlags().StringArrayP("port", "p", nil, "port to open")
	cmdInstanceCreate.PersistentFlags().StringArrayP("udp", "", nil, "udp ports to forward")
	cmdInstanceCreate.PersistentFlags().StringP("imagename", "i", "", "image name [required]")
	cmdInstanceCreate.PersistentFlags().StringP("flavor", "f", "", "flavor name for cloud provider")
	cmdInstanceCreate.PersistentFlags().StringP("domainname", "d", "", "domain name for instance")

	cmdInstanceCreate.MarkPersistentFlagRequired("imagename")
	return cmdInstanceCreate
}

func instanceCreateCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")
	flavor, _ := cmd.Flags().GetString("flavor")
	imagename, _ := cmd.Flags().GetString("imagename")
	domainname, _ := cmd.Flags().GetString("domainname")

	if projectID != "" {
		c.CloudConfig.ProjectID = projectID
	}

	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	if flavor != "" {
		c.CloudConfig.Flavor = flavor
	}

	if imagename != "" {
		c.CloudConfig.ImageName = imagename
	}

	if domainname != "" {
		c.RunConfig.DomainName = domainname
	}

	if len(args) > 0 {
		c.RunConfig.InstanceName = args[0]
	} else if c.RunConfig.InstanceName == "" {
		c.RunConfig.InstanceName = fmt.Sprintf("%v-%v",
			filepath.Base(c.CloudConfig.ImageName),
			strconv.FormatInt(time.Now().Unix(), 10),
		)
	}

	portsFlag, err := cmd.Flags().GetStringArray("port")
	if err != nil {
		panic(err)
	}
	ports, err := PrepareNetworkPorts(portsFlag)
	if err != nil {
		exitWithError(err.Error())
		return
	}

	udpPortsFlag, err := cmd.Flags().GetStringArray("udp")
	if err != nil {
		panic(err)
	}
	udpPorts, err := PrepareNetworkPorts(udpPortsFlag)
	if err != nil {
		exitWithError(err.Error())
		return
	}
	c.RunConfig.UDPPorts = udpPorts
	c.RunConfig.Ports = append(c.RunConfig.Ports, ports...)

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
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

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")

	if projectID != "" {
		c.CloudConfig.ProjectID = projectID
	}

	if zone != "" {
		c.CloudConfig.Zone = zone
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
	return cmdInstanceDelete
}

func instanceDeleteCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getInstanceCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")

	if projectID != "" {
		c.CloudConfig.ProjectID = projectID
	}
	if zone != "" {
		c.CloudConfig.Zone = zone
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

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone

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

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone

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

	projectID, _ := cmd.Flags().GetString("projectid")
	zone, _ := cmd.Flags().GetString("zone")

	watch, err := strconv.ParseBool(cmd.Flag("watch").Value.String())
	if err != nil {
		panic(err)
	}

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitForCmd(cmd, err.Error())
	}

	err = p.PrintInstanceLogs(ctx, args[0], watch)
	if err != nil {
		exitWithError(err.Error())
	}
}

func getInstanceCommandDefaultConfig(cmd *cobra.Command) (c *lepton.Config, err error) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c = lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err = mergeContainer.Merge(c)

	return
}
