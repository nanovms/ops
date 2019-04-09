package cmd

import (
	"fmt"
	"strconv"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func printManifestHandler(cmd *cobra.Command, args []string) {
	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	targetRoot, err := cmd.Flags().GetString("target-root")
	if err != nil {
		panic(err)
	}

	config, _ := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)
	c.Program = args[0]
	c.NightlyBuild = nightly
	prepareImages(c)
	c.TargetRoot = targetRoot
	m, err := api.BuildManifest(c)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(m.String())
}

func ManifestCommand() *cobra.Command {
	var config string
	var targetRoot string
	var nightly bool
	var cmdPrintConfig = &cobra.Command{
		Use:   "manifest [ELF file]",
		Short: "Print the manifest to console",
		Args:  cobra.MinimumNArgs(1),
		Run:   printManifestHandler,
	}
	cmdPrintConfig.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdPrintConfig.PersistentFlags().StringVarP(&targetRoot, "target-root", "r", "", "target root")
	cmdPrintConfig.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	return cmdPrintConfig
}
