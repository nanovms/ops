package cmd

import (
	"fmt"
	"os"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func instanceCommandHandler(cmd *cobra.Command, args []string) {
	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		fmt.Printf(api.ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\n")
		fmt.Printf(api.ErrorColor, "Follow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
		os.Exit(1)
	}
	provider, _ := cmd.Flags().GetString("target-cloud")

	projectID, err := cmd.Flags().GetString("projectid")
	if err != nil || len(projectID) == 0 {
		fmt.Printf(api.ErrorColor, "error: Not a valid ProjectID.\n")
		os.Exit(1)
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.CloudConfig.ImageName = strings.Trim(args[0], " ")
	c.CloudConfig.ProjectID = projectID

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	gcloud := p.(*api.GCloud)
	err = gcloud.CreateInstance(ctx)
	if err != nil {
		fmt.Println(err)
	}
}

// InstanceCommands provided instance related commands
func InstanceCommands() *cobra.Command {
	var projectID string
	var targetCloud string
	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"new"},
		Args:      cobra.OnlyValidArgs,
	}

	var cmdInstanceNew = &cobra.Command{
		Use:   "new [imagename]",
		Short: "new nanos instance",
		Args:  cobra.MinimumNArgs(1),
		Run:   instanceCommandHandler,
	}
	cmdInstanceNew.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstanceNew.PersistentFlags().StringVarP(&projectID, "projectid", "n", "", "ProjectID")
	cmdInstance.AddCommand(cmdInstanceNew)
	return cmdInstance
}
