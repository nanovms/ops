package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func imageCommandHandler(cmd *cobra.Command, args []string) {
	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		fmt.Printf(api.ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\n")
		fmt.Printf(api.ErrorColor, "Follow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
		os.Exit(1)
	}

	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)
	c.Program = args[0]

	// override config from command line
	if len(provider) > 0 {
		c.CloudConfig.Platform = provider
	}

	if len(c.CloudConfig.Platform) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please select on of the cloud platform in config. [onprem, gcp]")
		os.Exit(1)
	}

	if len(c.CloudConfig.ProjectID) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud projectid in config.\n")
		os.Exit(1)
	}

	if len(c.CloudConfig.BucketName) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud bucket in config.\n")
		os.Exit(1)
	}

	setDefaultImageName(cmd, c)

	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)
	prepareImages(c)

	archpath, err := p.BuildImage(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	gcloud := p.(*api.GCloud)
	err = gcloud.Storage.CopyToBucket(c, archpath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = gcloud.CreateImage(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		imageName := fmt.Sprintf("nanos-%v-image", filepath.Base(c.Program))
		fmt.Printf("gcp image '%s' created...\n", imageName)
	}
}

func ImageCommands() *cobra.Command {
	var targetCloud string
	var config string
	var cmdImageCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   imageCommandHandler,
	}

	cmdImageCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdImageCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")

	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create"},
		Args:      cobra.OnlyValidArgs,
	}
	return cmdImage
}
