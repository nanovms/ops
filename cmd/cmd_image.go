package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/onprem"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImageCommands provides image related command on GCP
func ImageCommands() *cobra.Command {
	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create", "list", "delete", "resize", "sync", "cp", "ls", "tree"},
		Args:      cobra.OnlyValidArgs,
	}

	PersistConfigCommandFlags(cmdImage.PersistentFlags())
	PersistProviderCommandFlags(cmdImage.PersistentFlags())

	cmdImage.AddCommand(imageCreateCommand())
	cmdImage.AddCommand(imageListCommand())
	cmdImage.AddCommand(imageDeleteCommand())
	cmdImage.AddCommand(imageResizeCommand())
	cmdImage.AddCommand(imageSyncCommand())
	cmdImage.AddCommand(imageCopyCommand())
	cmdImage.AddCommand(imageLsCommand())
	cmdImage.AddCommand(imageTreeCommand())

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
		Args:  cobra.MinimumNArgs(1),
	}

	cmdImageDelete.PersistentFlags().StringP("lru", "", "", "clean least recently used images with a time notation. Use \"1w\" notation to delete images older than one week. Other notation examples are 300d, 3w, 1m and 2y.")
	cmdImageDelete.PersistentFlags().BoolP("assume-yes", "", false, "clean images without waiting for confirmation")
	cmdImageDelete.PersistentFlags().BoolP("force", "", false, "force even if image is being used by instance")

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
	forceFlag, _ := cmd.Flags().GetBool("force")

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	// Check if image being used
	images, err := p.GetImages(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	imageMap := make(map[string]string)
	for _, name := range args {
		for _, img := range images {
			if img.Name == name {
				imageMap[name] = img.Path
				break
			}
		}
	}

	if !forceFlag {

		instances, err := p.GetInstances(ctx)
		if err != nil {
			exitWithError(err.Error())
		}

		if len(instances) > 0 {
			for imgName, imgPath := range imageMap {
				for _, is := range instances {
					var matchedImage bool
					if c.CloudConfig.Platform == onprem.ProviderName {
						matchedImage = (is.Image == imgPath)
					} else {
						matchedImage = (is.Image == imgName)
					}

					if matchedImage {
						fmt.Printf("image '%s' is being used\n", imgName)
						os.Exit(1)
					}
				}
			}
		}

	}

	imagesToDelete := []string{}

	if lru != "" {
		olderThanDate, err := SubtractTimeNotation(time.Now(), lru)
		if err != nil {
			exitWithError(fmt.Errorf("failed getting date from lru flag: %s", err).Error())
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
		log.Info("There are no images to delete")
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
			log.Error(err)
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

func imageCopyCommand() *cobra.Command {
	var cmdCopy = &cobra.Command{
		Use:   "cp <image_name> <src>... <dest>",
		Short: "copy files from image to local filesystem",
		Run:   imageCopyCommandHandler,
		Args:  cobra.MinimumNArgs(3),
	}
	flags := cmdCopy.PersistentFlags()
	flags.BoolP("recursive", "r", false, "copy directories recursively")
	flags.BoolP("dereference", "L", false, "always follow symbolic links in image")
	return cmdCopy
}

func imageCopyCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	reader := getLocalImageReader(flags, args)
	defer reader.Close()
	destPath := args[len(args)-1]
	var destDir bool
	fileInfo, err := os.Stat(destPath)
	if (err == nil) && fileInfo.IsDir() {
		destDir = true
	}
	if (len(args) > 3) && !destDir {
		exitWithError(fmt.Sprintf("Destination '%s' is not a directory", destPath))
	}
	recursive, _ := flags.GetBool("recursive")
	dereference, _ := flags.GetBool("dereference")
	for _, srcPath := range args[1 : len(args)-1] {
		fileInfo, err := reader.Stat(srcPath)
		if err != nil {
			log.Errorf("Invalid source '%s': %v", srcPath, err)
			continue
		}
		var dest string
		if destDir {
			dest = path.Join(destPath, path.Base(srcPath))
		} else {
			dest = destPath
		}
		switch fileInfo.Mode() {
		case os.ModeDir:
			if !recursive {
				log.Warnf("Omitting directory '%s'", srcPath)
				continue
			}
			err = imageCopyDir(reader, srcPath, dest, dereference)
		case 0, os.ModeSymlink:
			err = reader.CopyFile(srcPath, dest, true)
			if err != nil {
				err = fmt.Errorf("Cannot copy '%s' to '%s': %v", srcPath, dest, err)
			}
		}
		if err != nil {
			log.Error(err)
		}
	}
}

func imageCopyDir(reader *fs.Reader, src, dest string, dereference bool) error {
	dirEntries, err := reader.ReadDir(src)
	if err != nil {
		return fmt.Errorf("Cannot read directory '%s': %v", src, err)
	}
	if _, err = os.Stat(dest); os.IsNotExist(err) {
		if err = os.Mkdir(dest, 0755); err != nil {
			return fmt.Errorf("Cannot create directory '%s': %v", src, err)
		}
	}
	for _, entry := range dirEntries {
		srcPath := path.Join(src, entry.Name())
		destPath := path.Join(dest, entry.Name())
		switch entry.Mode() {
		case os.ModeDir:
			err = imageCopyDir(reader, srcPath, destPath, dereference)
		case 0, os.ModeSymlink:
			err = reader.CopyFile(srcPath, destPath, dereference)
			if err != nil {
				err = fmt.Errorf("Cannot copy '%s' to '%s': %v", srcPath, destPath, err)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func imageLsCommand() *cobra.Command {
	var cmdLs = &cobra.Command{
		Use:   "ls <image_name> <path>",
		Short: "list files and directories in image",
		Run:   imageLsCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmdLs.PersistentFlags()
	flags.BoolP("long-format", "l", false, "use a long listing format")
	return cmdLs
}

func imageLsCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	reader := getLocalImageReader(flags, args)
	defer reader.Close()
	var srcPath string
	if len(args) < 2 {
		srcPath = "/"
	} else if len(args) > 2 {
		exitForCmd(cmd, "Too many arguments")
	} else {
		srcPath = args[1]
	}
	longFormat, _ := flags.GetBool("long-format")
	fileInfo, err := reader.Stat(srcPath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot access '%s': %v", srcPath, err))
	}
	switch fileInfo.Mode() {
	case os.ModeDir:
		dirEntries, err := reader.ReadDir(srcPath)
		if err != nil {
			exitWithError(fmt.Sprintf("Cannot read directory '%s': %v", srcPath, err))
		}
		for index, entry := range dirEntries {
			if index > 0 {
				if longFormat {
					fmt.Println()
				} else {
					fmt.Print(" ")
				}
			}
			switch entry.Mode() {
			case os.ModeDir:
				imageLsDir(entry, longFormat)
			case os.ModeSymlink:
				imageLsSymlink(reader, path.Join(srcPath, entry.Name()), entry, longFormat)
			default: // regular file
				imageLsFile(entry, longFormat)
			}
		}
	case os.ModeSymlink:
		imageLsSymlink(reader, srcPath, fileInfo, longFormat)
	case 0: // regular file
		imageLsFile(fileInfo, longFormat)
	}
	fmt.Println()
}

func imageLsDir(fileInfo os.FileInfo, longFormat bool) {
	var infoString string
	if longFormat {
		infoString = fmt.Sprintf("\t %s %s%s%s/", fileInfo.ModTime().Format(time.ANSIC),
			log.ConsoleColors.Blue(), fileInfo.Name(), log.ConsoleColors.Reset())
	} else {
		infoString = log.ConsoleColors.Blue() + fileInfo.Name() + log.ConsoleColors.Reset()
	}
	fmt.Print(infoString)
}

func imageLsFile(fileInfo os.FileInfo, longFormat bool) {
	var infoString string
	if longFormat {
		infoString = fmt.Sprintf("%8d %s %s", fileInfo.Size(),
			fileInfo.ModTime().Format(time.ANSIC), fileInfo.Name())
	} else {
		infoString = fileInfo.Name()
	}
	fmt.Print(infoString)
}

func imageLsSymlink(reader *fs.Reader, filePath string, fileInfo os.FileInfo, longFormat bool) {
	var infoString string
	if longFormat {
		target, _ := reader.ReadLink(filePath)
		infoString = fmt.Sprintf("%8d %s %s%s%s -> %s", len(target),
			fileInfo.ModTime().Format(time.ANSIC), log.ConsoleColors.Cyan(), fileInfo.Name(),
			log.ConsoleColors.Reset(), target)
	} else {
		infoString = log.ConsoleColors.Cyan() + fileInfo.Name() + log.ConsoleColors.Reset()
	}
	fmt.Print(infoString)
}

func imageTreeCommand() *cobra.Command {
	var cmdLs = &cobra.Command{
		Use:   "tree <image_name>",
		Short: "display image filesystem contents in tree format",
		Run:   imageTreeCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdLs
}

func imageTreeCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	reader := getLocalImageReader(flags, args)
	dumpFSEntry(reader, "/", 0)
	reader.Close()
}

func dumpFSEntry(reader *fs.Reader, srcPath string, indent int) {
	fileInfo, err := reader.Stat(srcPath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot access '%s': %v", srcPath, err))
	}
	switch fileInfo.Mode() {
	case os.ModeDir:
		fmt.Println(getDumpLine(indent, fileInfo, log.ConsoleColors.Blue()))
		dirEntries, err := reader.ReadDir(srcPath)
		if err != nil {
			exitWithError(fmt.Sprintf("Cannot read directory '%s': %v", srcPath, err))
		}
		for _, entry := range dirEntries {
			dumpFSEntry(reader, path.Join(srcPath, entry.Name()), indent+1)
		}
	case os.ModeSymlink:
		fmt.Print(getDumpLine(indent, fileInfo, log.ConsoleColors.Cyan()))
		target, _ := reader.ReadLink(srcPath)
		fmt.Printf(" -> %s\n", target)
	case 0: // regular file
		fmt.Println(getDumpLine(indent, fileInfo, log.ConsoleColors.White()))
	}
}

func getDumpLine(indent int, fileInfo os.FileInfo, color string) string {
	line := ""
	for i := 0; i < indent; i++ {
		line += "|   "
	}
	line += color + fileInfo.Name() + log.ConsoleColors.Reset()
	return line
}

func getLocalImageReader(flags *pflag.FlagSet, args []string) *fs.Reader {
	c := lepton.NewConfig()
	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}
	if c.CloudConfig.Platform != "onprem" {
		exitWithError("Image subcommand not implemented yet for cloud images")
	}
	imageName := args[0]
	imagePath := path.Join(lepton.LocalImageDir, imageName)
	if _, err := os.Stat(imagePath); err != nil {
		if os.IsNotExist(err) && !strings.HasSuffix(imagePath, ".img") {
			imagePath += ".img"
			_, err = os.Stat(imagePath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				exitWithError(fmt.Sprintf("Local image %s not found", imageName))
			} else {
				exitWithError(fmt.Sprintf("Cannot read image %s: %v", imageName, err))
			}
		}
	}
	reader, err := fs.NewReader(imagePath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot load image %s: %v", imageName, err))
	}
	return reader
}
