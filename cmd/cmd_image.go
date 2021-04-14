package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
)

// ImageCommands provides image related command on GCP
func ImageCommands() *cobra.Command {
	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create", "list", "delete", "resize", "sync"},
		Args:      cobra.OnlyValidArgs,
	}

	PersistConfigCommandFlags(cmdImage.PersistentFlags())
	PersistProviderCommandFlags(cmdImage.PersistentFlags())

	cmdImage.AddCommand(imageCreateCommand())
	cmdImage.AddCommand(imageListCommand())
	cmdImage.AddCommand(imageDeleteCommand())
	cmdImage.AddCommand(imageResizeCommand())
	cmdImage.AddCommand(imageSyncCommand())

	return cmdImage
}

func imageCreateCommand() *cobra.Command {

	var cmdImageCreate = &cobra.Command{
		Use:   "create [elf|program]",
		Short: "create nanos image from ELF",
		Run:   imageCreateCommandHandler,
	}

	persistentFlags := cmdImageCreate.PersistentFlags()

	PersistBuildImageCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)
	PersistPkgCommandFlags(persistentFlags)

	return cmdImageCreate
}

func imageCreateCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	c := lepton.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	if len(args) > 0 {
		c.Program = args[0]
	}

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, providerFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	if c.Program == "" {
		exitForCmd(cmd, "no program specified")
	}

	if len(c.CloudConfig.BucketName) == 0 &&
		(c.CloudConfig.Platform == "gcp" ||
			c.CloudConfig.Platform == "aws" ||
			c.CloudConfig.Platform == "azure") {
		exitWithError("Please specify a cloud bucket in config")
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	var keypath string
	if pkgFlags.Package != "" {
		keypath, err = p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
		if err != nil {
			exitWithError(err.Error())
		}
	} else {
		keypath, err = p.BuildImage(ctx)
		if err != nil {
			exitWithError(err.Error())
		}
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		exitWithError(err.Error())
	}

	imageName := c.CloudConfig.ImageName
	if c.CloudConfig.Platform == "onprem" {
		imageName = c.RunConfig.Imagename
	}

	fmt.Printf("%s image '%s' created...\n", c.CloudConfig.Platform, imageName)
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
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

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
	cmdImageDelete.PersistentFlags().BoolP("assume-yes", "", false, "clean images without waiting for confirmation")

	return cmdImageDelete
}

func imageDeleteCommandHandler(cmd *cobra.Command, args []string) {

	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	lru, _ := cmd.Flags().GetString("lru")
	assumeYes, _ := cmd.Flags().GetBool("assume-yes")

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
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

	if assumeYes != true {
		fmt.Printf("You are about to delete the next images:\n")
		for _, i := range imagesToDelete {
			fmt.Println(i)
		}
		fmt.Println("Are you sure? (yes/no)")
		confirmation := askForConfirmation()
		if !confirmation {
			return
		}
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

	c := &types.Config{}

	err := unWarpConfig(config, c)
	if err != nil {
		exitWithError(err.Error())
	}

	globalFlags := NewGlobalCommandFlags(cmd.Flags())
	err = globalFlags.MergeToConfig(c)
	if err != nil {
		exitWithError(err.Error())
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		c.CloudConfig.Zone = zone
	}

	targetCloud, _ := cmd.Flags().GetString("target-cloud")
	p, err := provider.CloudProvider(targetCloud, &c.CloudConfig)
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
	conf := &types.Config{}
	err := unWarpConfig(config, conf)
	if err != nil {
		exitWithError(err.Error())
	}

	globalFlags := NewGlobalCommandFlags(cmd.Flags())
	err = globalFlags.MergeToConfig(conf)
	if err != nil {
		exitWithError(err.Error())
	}

	zone, _ := cmd.Flags().GetString("zone")
	if zone != "" {
		conf.CloudConfig.Zone = zone
	}

	src, err := provider.CloudProvider(source, &conf.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}

	target, _ := cmd.Flags().GetString("target-cloud")
	tar, err := provider.CloudProvider(target, &conf.CloudConfig)
	if err != nil {
		exitWithError(err.Error())
	}

	err = src.SyncImage(conf, tar, image)
	if err != nil {
		exitWithError(err.Error())
	}
}
