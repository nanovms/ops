package cmd

import (
	"fmt"
	"os"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func buildCommandHandler(cmd *cobra.Command, args []string) {
	targetRoot, _ := cmd.Flags().GetString("target-root")
	provider, _ := cmd.Flags().GetString("target-cloud")

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	c := unWarpConfig(config)
	c.Program = args[0]
	c.TargetRoot = targetRoot
	setDefaultImageName(cmd, c)

	p := getCloudProvider(provider)

	ctx := api.NewContext(c, &p)
	prepareImages(c)
	if _, err := p.BuildImage(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Bootable image file:%s\n", c.RunConfig.Imagename)
}

// BuildCommand helps you to build image from ELF
func BuildCommand() *cobra.Command {
	var config string
	var targetRoot string
	var targetCloud string
	var imageName string

	var cmdBuild = &cobra.Command{
		Use:   "build [ELF file]",
		Short: "Build an image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   buildCommandHandler,
	}

	cmdBuild.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdBuild.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")
	cmdBuild.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform[gcp, onprem]")
	cmdBuild.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdBuild
}
