package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	os.Exit(1)
}

func exitForCmd(cmd *cobra.Command, errs string) {
	fmt.Println(fmt.Sprintf(api.ErrorColor, errs))
	cmd.Help()
	os.Exit(1)
}

func instanceCreateCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	var c *api.Config
	if config != "" {
		c = unWarpConfig(config)
	} else {
		c = &api.Config{}
	}

	projectID, _ := cmd.Flags().GetString("projectid")
	if projectID != "" {
		c.CloudConfig.ProjectID = projectID
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	flavor, _ := cmd.Flags().GetString("flavor")
	if flavor != "" {
		c.CloudConfig.Flavor = flavor
	}

	imagename, _ := cmd.Flags().GetString("imagename")
	c.CloudConfig.ImageName = imagename

	ports := []int{}
	port, err := cmd.Flags().GetStringArray("port")
	if err != nil {
		panic(err)
	}

	for _, p := range port {
		i, err := strconv.Atoi(p)
		if err != nil {
			panic(err)
		}

		ports = append(ports, i)
	}

	initDefaultRunConfigs(c, ports)

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	err = p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceCreateCommand() *cobra.Command {
	var imageName, config, flavor string

	var cmdInstanceCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos instance",
		Run:   instanceCreateCommandHandler,
	}

	cmdInstanceCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "config for nanos")
	cmdInstanceCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name [required]")
	cmdInstanceCreate.PersistentFlags().StringVarP(&flavor, "flavor", "f", "g1-small", "flavor name for GCP")

	cmdInstanceCreate.MarkPersistentFlagRequired("imagename")
	return cmdInstanceCreate
}

func instanceListCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")

	p := getCloudProvider(provider)
	c := api.Config{}

	projectID, _ := cmd.Flags().GetString("projectid")
	if projectID == "" && provider == "gcp" {
		exitForCmd(cmd, "projectid argument missing")
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitForCmd(cmd, "zone argument missing")
	}

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err := p.ListInstances(ctx)
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

func instanceDeleteCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	c := api.Config{}

	projectID, _ := cmd.Flags().GetString("projectid")

	if projectID == "" && provider == "gcp" {
		exitForCmd(cmd, "projectid argument missing")
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitForCmd(cmd, "zone argument missing")
	}
	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err := p.DeleteInstance(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceStartCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	c := api.Config{}

	projectID, _ := cmd.Flags().GetString("projectid")

	if projectID == "" && provider == "gcp" {
		exitForCmd(cmd, "projectid argument missing")
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitForCmd(cmd, "zone argument missing")
	}

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err := p.StartInstance(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceStopCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	c := api.Config{}

	projectID, _ := cmd.Flags().GetString("projectid")

	if projectID == "" && provider == "gcp" {
		exitForCmd(cmd, "projectid argument missing")
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitForCmd(cmd, "zone argument missing")
	}

	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err := p.StopInstance(ctx, args[0])
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

func instanceStopCommand() *cobra.Command {
	var cmdInstanceStop = &cobra.Command{
		Use:   "stop <instance_name>",
		Short: "stop instance on provider",
		Run:   instanceStopCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdInstanceStop
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

func instanceLogsCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	c := api.Config{}
	projectID, _ := cmd.Flags().GetString("projectid")
	if projectID == "" {
		exitForCmd(cmd, "projectid argument missing")
	}
	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitForCmd(cmd, "zone argument missing")
	}
	watch, err := strconv.ParseBool(cmd.Flag("watch").Value.String())
	if err != nil {
		panic(err)
	}
	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err = p.GetInstanceLogs(ctx, args[0], watch)
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

// InstanceCommands provided instance related commands
func InstanceCommands() *cobra.Command {
	var targetCloud, projectID, zone string
	var ports []string

	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"create", "list", "delete", "stop", "start", "logs"},
		Args:      cobra.OnlyValidArgs,
	}

	cmdInstance.PersistentFlags().StringArrayVarP(&ports, "port", "p", nil, "port to open")
	cmdInstance.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, aws, onprem, vultr, vsphere]")
	cmdInstance.PersistentFlags().StringVarP(&projectID, "projectid", "g", os.Getenv("GOOGLE_CLOUD_PROJECT"), "project-id for GCP or set env GOOGLE_CLOUD_PROJECT")
	cmdInstance.PersistentFlags().StringVarP(&zone, "zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for GCP or set env GOOGLE_CLOUD_ZONE")
	cmdInstance.AddCommand(instanceCreateCommand())
	cmdInstance.AddCommand(instanceListCommand())
	cmdInstance.AddCommand(instanceDeleteCommand())
	cmdInstance.AddCommand(instanceStopCommand())
	cmdInstance.AddCommand(instanceStartCommand())
	cmdInstance.AddCommand(instanceLogsCommand())

	return cmdInstance
}
