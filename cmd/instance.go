package cmd

import (
	"fmt"
	"os"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func exitWithError(errs string) {
	fmt.Printf(api.ErrorColor, errs)
	os.Exit(1)
}

func instanceCreateCommandHandler(cmd *cobra.Command, args []string) {
	checkCredentenaisProvided()
	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	imagename, _ := cmd.Flags().GetString("imagename")
	if imagename == "" {
		exitWithError("imagename argument missing")
	}

	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.CloudConfig.ImageName = imagename

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	err := p.CreateInstance(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func instanceCreateCommand() *cobra.Command {
	var targetCloud, imageName string
	var cmdInstanceCreate = &cobra.Command{
		Use:   "create [imagename]",
		Short: "create nanos instance",
		Run:   instanceCreateCommandHandler,
	}
	cmdInstanceCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdInstanceCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdInstanceCreate
}

// InstanceCommands provided instance related commands
func InstanceCommands() *cobra.Command {
	var cmdInstance = &cobra.Command{
		Use:       "instance",
		Short:     "manage nanos instances",
		ValidArgs: []string{"create"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdInstance.AddCommand(instanceCreateCommand())
	return cmdInstance
}
