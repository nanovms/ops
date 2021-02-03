package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

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
	cmdImage.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "onprem", "cloud platform [gcp, aws, do, vultr, onprem, hyper-v, upcloud]")
	cmdImage.PersistentFlags().StringVarP(&zone, "zone", "z", os.Getenv("GOOGLE_CLOUD_ZONE"), "zone name for target cloud platform. defaults to GCP or set env GOOGLE_CLOUD_ZONE")
	cmdImage.AddCommand(imageCreateCommand())
	cmdImage.AddCommand(imageListCommand())
	cmdImage.AddCommand(imageDeleteCommand())
	cmdImage.AddCommand(imageResizeCommand())
	cmdImage.AddCommand(imageSyncCommand())
	return cmdImage
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

	if len(c.CloudConfig.BucketName) == 0 && c.CloudConfig.Platform != "onprem" && c.CloudConfig.Platform != "hyper-v" && c.CloudConfig.Platform != "upcloud" {
		exitWithError("Please specify a cloud bucket in config")
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

	p, ctx, err := getProviderAndContext(c, provider)
	if err != nil {
		exitWithError(err.Error())
	}

	var keypath string
	if len(pkg) > 0 {
		c.Args = append(c.Args, cmdargs...)

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
		ctx = api.NewContext(c)

		keypath, err = p.BuildImageWithPackage(ctx, expackage)
		if err != nil {
			exitWithError(err.Error())
		}
	} else {
		if len(cmdargs) != 0 {
			c.Program = cmdargs[0]
		} else if len(c.Args) != 0 {
			c.Program = c.Args[0]
		} else {
			exitWithError("Please mention program to run")
		}

		setDefaultImageName(cmd, c)
		keypath, err = p.BuildImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		}
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		exitWithError(err.Error())
	}

	fmt.Printf("%s image '%s' created...\n", provider, c.CloudConfig.ImageName)
}

func imageListCommand() *cobra.Command {
	var cmdImageList = &cobra.Command{
		Use:   "list",
		Short: "list images from provider",
		Run:   imageListCommandHandler,
	}
	return cmdImageList
}

func imageListCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	var c *api.Config
	c = unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	p, err := getCloudProvider(provider, &c.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}

	ctx := api.NewContext(c)

	err = p.ListImages(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func imageDeleteCommand() *cobra.Command {
	var cmdImageDelete = &cobra.Command{
		Use:   "delete <image_name>",
		Short: "delete images from provider",
		Run:   imageDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(0),
	}

	cmdImageDelete.PersistentFlags().StringP("lru", "", "", "clean least recently used images with a time notation. Use \"1w\" notation to delete images older than one week. Other notation examples are 300d, 3w, 1m and 2y.")

	return cmdImageDelete
}

func imageDeleteCommandHandler(cmd *cobra.Command, args []string) {
	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	lru, _ := cmd.Flags().GetString("lru")
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	p, ctx, err := getProviderAndContext(c, provider)
	if err != nil {
		exitWithError(err.Error())
	}

	imagesToDelete := []string{}

	if lru != "" {
		olderThanDate, err := SubtractTimeNotation(time.Now(), lru)
		if err != nil {
			exitWithError(fmt.Errorf("failed getting date from lru flag: %s", err).Error())
		}

		images, err := p.GetImages(ctx)
		if err != nil {
			exitWithError(err.Error())
		}

		for _, image := range images {
			if image.Created.Before(olderThanDate) {
				if image.ID != "" {
					imagesToDelete = append(imagesToDelete, image.ID)
				} else {
					imagesToDelete = append(imagesToDelete, image.Name)
				}
			}
		}
	}

	if len(args) > 0 {
		imagesToDelete = append(imagesToDelete, args...)
	}

	if len(imagesToDelete) == 0 {
		fmt.Println("There are no images to delete")
		return
	}

	fmt.Printf("You are about to delete the next images:\n")
	for _, i := range imagesToDelete {
		fmt.Println(i)
	}
	fmt.Println("Are you sure? (yes/no)")
	confirmation := askForConfirmation()
	if !confirmation {
		return
	}

	responses := make(chan error)

	deleteImage := func(imageName string) {
		errMsg := p.DeleteImage(ctx, imageName)
		if errMsg != nil {
			errMsg = fmt.Errorf("failed deleting %s: %v", imageName, errMsg)
		}

		responses <- errMsg
	}

	for _, i := range imagesToDelete {
		go deleteImage(i)
	}

	for range imagesToDelete {
		err = <-responses
		if err != nil {
			fmt.Println(err)
		}
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

// only targets local images today
func imageResizeCommandHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)

	c := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), c)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	provider, _ := cmd.Flags().GetString("target-cloud")
	p, err := getCloudProvider(provider, &c.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}
	ctx := api.NewContext(c)

	err = p.ResizeImage(ctx, args[0], args[1])
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
	cmdImageSync.PersistentFlags().StringVarP(&sourceCloud, "source-cloud", "s", "onprem", "cloud platform [gcp, aws, do, vultr, onprem, hyper-v, upcloud]")
	return cmdImageSync
}

func imageSyncCommandHandler(cmd *cobra.Command, args []string) {
	image := args[0]
	// TODO only accepts onprem for now, implement for other source providers later
	source, _ := cmd.Flags().GetString("source-cloud")
	if source != "onprem" {
		exitWithError(source + " sync not yet implemented")
	}

	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	AppendGlobalCmdFlagsToConfig(cmd.Flags(), conf)

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		conf.CloudConfig.Zone = zone
	}

	src, err := getCloudProvider(source, &conf.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}

	target, _ := cmd.Flags().GetString("target-cloud")
	tar, err := getCloudProvider(target, &conf.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}

	err = src.SyncImage(conf, tar, image)
	if err != nil {
		exitWithError(err.Error())
	}
}
