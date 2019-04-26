package cmd

import (
	"fmt"
	"os"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Println(errs)
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
	imagename, _ := cmd.Flags().GetString("imagename")
	c.CloudConfig.ImageName = imagename

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	err := p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceCreateCommand() *cobra.Command {
	var imageName, config string
	var cmdInstanceCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos instance",
		Run:   instanceCreateCommandHandler,
	}
	cmdInstanceCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "config for nanos")
	cmdInstanceCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name [required]")
	cmdInstanceCreate.MarkPersistentFlagRequired("imagename")
	return cmdInstanceCreate
}

func instanceListCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	c := api.Config{}
	projectID, _ := cmd.Flags().GetString("projectid")
	if projectID == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "projectid argument missing"))
	}
	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "zone argument missing"))
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
	if projectID == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "projectid argument missing"))
	}
	zone, _ := cmd.Flags().GetString("zone")
	if zone == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "zone argument missing"))
	}
	c.CloudConfig.ProjectID = projectID
	c.CloudConfig.Zone = zone
	ctx := api.NewContext(&c, &p)
	err := p.DeleteInstance(ctx, args[0])
	if err != nil {
		exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
	}
}

func instanceDeleteCommand() *cobra.Command {
	var cmdInstanceDelete = &cobra.Command{
		Use:   "delete",
		Short: "delete instance on provider",
		Run:   instanceDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdInstanceDelete
}

// InstanceCommands provided instance related commands
func InstanceCommands() *cobra.Command {
	var targetCloud, projectID, zone string
	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"create", "list", "delete"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdInstance.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstance.PersistentFlags().StringVarP(&projectID, "projectid", "p", os.Getenv("GOOGLE_CLOUD_PROJECT"), "project-id for GCP or set env GOOGLE_CLOUD_PROJECT")
	cmdInstance.PersistentFlags().StringVarP(&zone, "zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for GCP or set env GOOGLE_CLOUD_ZONE")
	cmdInstance.AddCommand(instanceCreateCommand())
	cmdInstance.AddCommand(instanceListCommand())
	cmdInstance.AddCommand(instanceDeleteCommand())
	return cmdInstance
}
