package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
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
	mounts, _ := cmd.Flags().GetStringArray("mounts")

	nightly, err := strconv.ParseBool(cmd.Flag("nightly").Value.String())
	if err != nil {
		panic(err)
	}

	c := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	// override config from command line
	if len(provider) > 0 {
		c.CloudConfig.Platform = provider
	}

	if nightly {
		c.NightlyBuild = nightly
	}

	if c.CloudConfig.Platform == "azure" {
		c.RunConfig.Klibs = append(c.RunConfig.Klibs, "cloud_init")
	}

	if len(c.CloudConfig.Platform) == 0 {
		exitWithError("Please select on of the cloud platform in config. [onprem, aws, gcp, do, vsphere, vultr]")
	}

	if c.CloudConfig.Platform == "gcp" && len(c.CloudConfig.ProjectID) == 0 {
		exitWithError("Please specify a cloud projectid in config")
	}

	if len(c.CloudConfig.BucketName) == 0 {
		exitWithError("Please specify a cloud bucket in config")
	}

	if len(pkg) > 0 {
		c.Args = append(c.Args, cmdargs...)
	} else {
		if len(cmdargs) != 0 {
			c.Program = cmdargs[0]
		} else if len(c.Args) != 0 {
			c.Program = c.Args[0]
		} else {
			exitWithError("Please mention program to run")
		}
	}

	prepareImages(c)
	// borrow BuildDir from config
	bd := c.BuildDir
	c.BuildDir = api.LocalVolumeDir
	err = api.AddMounts(mounts, c)
	if err != nil {
		exitWithError(err.Error())
	}
	c.BuildDir = bd

	p, err := getCloudProvider(provider)
	if err != nil {
		exitWithError(err.Error())
	}
	ctx := api.NewContext(c, &p)

	var keypath string
	if len(pkg) > 0 {
		expackage := downloadAndExtractPackage(pkg)

		// load the package manifest
		manifest := path.Join(expackage, "package.manifest")
		if _, err := os.Stat(manifest); err != nil {
			exitWithError(err.Error())
		}

		pkgConfig := unWarpConfig(manifest)
		c = mergeConfigs(pkgConfig, c)
		setDefaultImageName(cmd, c)

		// Config merged with package config, need to update context
		ctx = api.NewContext(c, &p)

		keypath, err = p.BuildImageWithPackage(ctx, expackage)

	} else {
		setDefaultImageName(cmd, c)
		keypath, err = p.BuildImage(ctx)
	}

	if err != nil {
		exitWithError(err.Error())
	}

	if c.CloudConfig.Platform == "vultr" {
		do := p.(*api.Vultr)
		err = do.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = do.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("do image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "do" {
		do := p.(*api.DigitalOcean)
		err = do.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = do.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("do image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "gcp" {
		gcloud := p.(*api.GCloud)
		err = gcloud.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = gcloud.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("gcp image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "aws" {
		aws := p.(*api.AWS)

		// verify we can even use the vm importer
		api.VerifyRole(ctx, c.CloudConfig.BucketName)

		err = aws.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = aws.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("aws image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "vsphere" {
		vsphere := p.(*api.Vsphere)
		err = vsphere.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = vsphere.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("vsphere image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "openstack" {
		os := p.(*api.OpenStack)

		err = os.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("openstack image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

	if c.CloudConfig.Platform == "azure" {
		azure := p.(*api.Azure)

		err = azure.Storage.CopyToBucket(c, keypath)
		if err != nil {
			exitWithError(err.Error())
		}

		err = azure.CreateImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("azure image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}

}

func imageCreateCommand() *cobra.Command {
	var (
		config, pkg, imageName string
		args, mounts           []string
		nightly                bool
	)

	var cmdImageCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos image from ELF",
		Run:   imageCreateCommandHandler,
	}

	cmdImageCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdImageCreate.PersistentFlags().StringVarP(&pkg, "package", "p", "", "ops package name")
	cmdImageCreate.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdImageCreate.PersistentFlags().StringArrayVar(&mounts, "mounts", nil, "mount <volume_id:mount_path>")
	cmdImageCreate.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")

	cmdImageCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")
	return cmdImageCreate
}

// only targets local images today
func imageResizeCommandHandler(cmd *cobra.Command, args []string) {

	provider, _ := cmd.Flags().GetString("target-cloud")
	p, err := getCloudProvider(provider)
	if err != nil {
		exitWithError(err.Error())
	}

	var c *api.Config
	c = api.NewConfig()
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	ctx := api.NewContext(c, &p)

	err = p.ResizeImage(ctx, args[0], args[1])
	if err != nil {
		exitWithError(err.Error())
	}
}

func imageResizeCommand() *cobra.Command {
	var cmdImageResize = &cobra.Command{
		Use:   "resize <image_name> <new_size>",
		Short: "resize image",
		Run:   imageResizeCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdImageResize
}

func imageListCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")

	var c *api.Config
	c = api.NewConfig()
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	p, err := getCloudProvider(provider)
	if err != nil {
		exitWithError(err.Error())
	}

	ctx := api.NewContext(c, &p)

	err = p.ListImages(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func imageListCommand() *cobra.Command {
	var cmdImageList = &cobra.Command{
		Use:   "list",
		Short: "list images from provider",
		Run:   imageListCommandHandler,
	}
	return cmdImageList
}

func imageDeleteCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	p, err := getCloudProvider(provider)
	if err != nil {
		exitWithError(err.Error())
	}

	var c *api.Config
	c = api.NewConfig()
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	ctx := api.NewContext(c, &p)

	err = p.DeleteImage(ctx, args[0])
	if err != nil {
		exitWithError(err.Error())
	}

	if c.CloudConfig.Platform == "azure" || provider == "azure" {
		azure := p.(*api.Azure)

		err = azure.Storage.DeleteFromBucket(c, args[0]+".vhd")
		if err != nil {
			exitWithError(err.Error())
		} else {
			fmt.Printf("\nImage %s deleted successfully", args[0])
		}
	}
}

func imageDeleteCommand() *cobra.Command {
	var cmdImageDelete = &cobra.Command{
		Use:   "delete <image_name>",
		Short: "delete images from provider",
		Run:   imageDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdImageDelete
}

func imageSyncCommandHandler(cmd *cobra.Command, args []string) {
	image := args[0]
	// TODO only accepts onprem for now, implement for other source providers later
	source, _ := cmd.Flags().GetString("source-cloud")
	if source != "onprem" {
		exitWithError(source + " sync not yet implemented")
	}
	src, err := getCloudProvider(source)
	if err != nil {
		exitWithError(err.Error())
	}

	target, _ := cmd.Flags().GetString("target-cloud")
	tar, err := getCloudProvider(target)
	if err != nil {
		exitWithError(err.Error())
	}

	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), conf)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		conf.CloudConfig.Zone = zone
	}

	err = src.SyncImage(conf, tar, image)
	if err != nil {
		exitWithError(err.Error())
	}
}

func imageSyncCommand() *cobra.Command {
	var sourceCloud string
	var cmdImageSync = &cobra.Command{
		Use:   "sync <image_name>",
		Short: "sync image with from one provider to another",
		Run:   imageSyncCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdImageSync.PersistentFlags().StringVarP(&sourceCloud, "source-cloud", "s", "onprem", "cloud platform [gcp, aws, do, vultr, onprem]")
	return cmdImageSync
}

// ImageCommands provides image related command on GCP
func ImageCommands() *cobra.Command {
	var config, targetCloud, zone string
	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create", "list", "delete", "resize", "sync"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdImage.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdImage.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "onprem", "cloud platform [gcp, aws, do, vultr, onprem]")
	cmdImage.PersistentFlags().StringVarP(&zone, "zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for target cloud platform. defaults to GCP or set env GOOGLE_CLOUD_ZONE")
	cmdImage.AddCommand(imageCreateCommand())
	cmdImage.AddCommand(imageListCommand())
	cmdImage.AddCommand(imageDeleteCommand())
	cmdImage.AddCommand(imageResizeCommand())
	cmdImage.AddCommand(imageSyncCommand())
	return cmdImage
}
