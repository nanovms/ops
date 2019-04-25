package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func imageCreateCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	pkg, _ := cmd.Flags().GetString("package")
	pkg = strings.TrimSpace(pkg)
	cmdargs, _ := cmd.Flags().GetStringArray("args")

	c := unWarpConfig(config)
	// override config from command line
	if len(provider) > 0 {
		c.CloudConfig.Platform = provider
	}

	if len(c.CloudConfig.Platform) == 0 {
		exitWithError(fmt.Sprintf(api.ErrorColor, "Please select on of the cloud platform in config. [onprem, gcp]"))
	}

	if len(c.CloudConfig.ProjectID) == 0 {
		exitWithError(fmt.Sprintf(api.ErrorColor, "Please specifiy a cloud projectid in config"))
	}

	if len(c.CloudConfig.BucketName) == 0 {
		exitWithError(fmt.Sprintf(api.ErrorColor, "Please specifiy a cloud bucket in config"))
	}

	if len(pkg) > 0 {
		c.Args = append(c.Args, cmdargs...)
	} else {
		if len(cmdargs) != 0 {
			c.Program = cmdargs[0]
		} else if len(c.Args) != 0 {
			c.Program = c.Args[0]
		} else {
			exitWithError(fmt.Sprintf(api.ErrorColor, "Please mention program to run"))
		}
	}

	prepareImages(c)
	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)

	var archpath string
	var err error
	if len(pkg) > 0 {
		expackage := downloadAndExtractPackage(pkg)
		// load the package manifest
		manifest := path.Join(expackage, "package.manifest")
		if _, err := os.Stat(manifest); err != nil {
			exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
		}
		pkgConfig := unWarpConfig(manifest)
		c = mergeConfigs(pkgConfig, c)
		setDefaultImageName(cmd, c)
		// Config merged with package config, need to update context
		ctx = api.NewContext(c, &p)
		archpath, err = p.BuildImageWithPackage(ctx, expackage)
	} else {
		setDefaultImageName(cmd, c)
		archpath, err = p.BuildImage(ctx)
	}

	if err != nil {
		exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
	}

	if c.CloudConfig.Platform == "gcp" {
		gcloud := p.(*api.GCloud)
		err = gcloud.Storage.CopyToBucket(c, archpath)
		if err != nil {
			exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
		}
		err = gcloud.CreateImage(ctx)
		if err != nil {
			exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
		} else {
			fmt.Printf("gcp image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}
}

func imageCreateCommand() *cobra.Command {
	var (
		targetCloud, config, pkg, imageName string
		args                                []string
	)
	var cmdImageCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos image from ELF",
		Run:   imageCreateCommandHandler,
	}
	cmdImageCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdImageCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdImageCreate.PersistentFlags().StringVarP(&pkg, "package", "p", "", "ops package name")
	cmdImageCreate.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdImageCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdImageCreate
}

func imageListCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p := getCloudProvider(provider)
	err := p.ListImages()
	if err != nil {
		exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
	}
}

func imageListCommand() *cobra.Command {
	var (
		targetCloud string
	)
	var cmdImageList = &cobra.Command{
		Use:   "list",
		Short: "list images from provider",
		Run:   imageListCommandHandler,
	}
	cmdImageList.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	return cmdImageList
}

func imageDeleteCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	imageName, _ := cmd.Flags().GetString("imagename")
	if imageName == "" {
		exitWithError(fmt.Sprintf(api.ErrorColor, "Please provide image name with -i option"))
	}
	p := getCloudProvider(provider)
	err := p.DeleteImage(imageName)
	if err != nil {
		exitWithError(fmt.Sprintf(api.ErrorColor, err.Error()))
	}
}

func imageDeleteCommand() *cobra.Command {
	var (
		targetCloud, imageName string
	)
	var cmdImageDelete = &cobra.Command{
		Use:   "delete",
		Short: "Delete images from provider",
		Run:   imageDeleteCommandHandler,
	}
	cmdImageDelete.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdImageDelete.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdImageDelete
}

// ImageCommands provides image related command on GCP
func ImageCommands() *cobra.Command {
	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdImage.AddCommand(imageCreateCommand())
	cmdImage.AddCommand(imageListCommand())
	cmdImage.AddCommand(imageDeleteCommand())
	return cmdImage
}
