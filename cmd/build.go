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

	cmdenvs, err := cmd.Flags().GetStringArray("envs")
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)

	c.Program = args[0]
	c.TargetRoot = targetRoot
	
	if len(cmdenvs) > 0 {
		if len(c.Env) == 0 {
			c.Env = make(map[string]string)
		}

		for i := 0; i < len(cmdenvs); i++ {
			ez := strings.Split(cmdenvs[i], "=")
			c.Env[ez[0]] = ez[1]
		}
	}

	setDefaultImageName(cmd, c)

	p, err := getCloudProvider(provider)
	if err != nil {
		exitWithError(err.Error())
	}

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
	var envs []string

	var cmdBuild = &cobra.Command{
		Use:   "build [ELF file]",
		Short: "Build an image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   buildCommandHandler,
	}

	cmdBuild.PersistentFlags().StringArrayVarP(&envs, "envs", "e", nil, "env arguments")
	cmdBuild.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdBuild.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")
	cmdBuild.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform[gcp, onprem]")
	cmdBuild.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdBuild
}
