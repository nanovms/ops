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
	imagename, _ := cmd.Flags().GetString("imagename")
	if imagename == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "imagename argument missing"))
	}

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

	c.CloudConfig.ImageName = imagename

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	err := p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceCreateCommand() *cobra.Command {
	var targetCloud, imageName, projectID, zone, config string
	var cmdInstanceCreate = &cobra.Command{
		Use:   "create [imagename]",
		Short: "create nanos instance",
		Run:   instanceCreateCommandHandler,
	}
	cmdInstanceCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstanceCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "config for nanos")
	cmdInstanceCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	cmdInstanceCreate.PersistentFlags().StringVarP(&projectID, "projectid", "p", "", "project-id for GCP")
	cmdInstanceCreate.PersistentFlags().StringVarP(&zone, "zone", "z", "", "zone name for GCP")
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
	var targetCloud, zone, projectID string
	var cmdInstanceList = &cobra.Command{
		Use:   "list",
		Short: "list instance on provider",
		Run:   instanceListCommandHandler,
	}
	cmdInstanceList.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstanceList.PersistentFlags().StringVarP(&projectID, "projectid", "p", "", "project-id for GCP")
	cmdInstanceList.PersistentFlags().StringVarP(&zone, "zone", "z", "", "zone name for GCP")
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
	var targetCloud, zone, projectID string
	var cmdInstanceDelete = &cobra.Command{
		Use:   "delete",
		Short: "delete instance on provider",
		Run:   instanceDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdInstanceDelete.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstanceDelete.PersistentFlags().StringVarP(&projectID, "projectid", "p", "", "project-id for GCP")
	cmdInstanceDelete.PersistentFlags().StringVarP(&zone, "zone", "z", "", "zone name for GCP")
	return cmdInstanceDelete
}

// InstanceCommands provided instance related commands
func InstanceCommands() *cobra.Command {
	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"create", "list", "delete"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdInstance.AddCommand(instanceCreateCommand())
	cmdInstance.AddCommand(instanceListCommand())
	cmdInstance.AddCommand(instanceDeleteCommand())
	return cmdInstance
}
