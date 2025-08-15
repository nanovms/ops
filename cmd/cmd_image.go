package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"encoding/json"

	"github.com/nanovms/ops/fs"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImageCommands provides image related command on GCP
func ImageCommands() *cobra.Command {
	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create", "list", "delete", "resize", "sync", "cat", "cp", "ls", "search", "tree", "env", "mirror"},
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
	cmdImage.AddCommand(imageCatCommand())
	cmdImage.AddCommand(imageLsCommand())
	cmdImage.AddCommand(imageTreeCommand())
	cmdImage.AddCommand(imageEnvCommand())
	cmdImage.AddCommand(imageMirrorCommand())
	cmdImage.AddCommand(imageSearchCommand())

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
	c := api.NewConfig()

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
	if c.CloudConfig.Platform == onprem.ProviderName {
		imageName = c.RunConfig.ImageName
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

func imageSearchCommand() *cobra.Command {
	var cmdSearchList = &cobra.Command{
		Use:   "search [imagename]",
		Short: "search images from provider",
		Run:   imageSearchCommandHandler,
	}
	return cmdSearchList
}

func imageSearchCommandHandler(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("search requires a regex to search an image for")
		os.Exit(1)
	}

	q := args[0]
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	err = p.ListImages(ctx, q)
	if err != nil {
		exitWithError(err.Error())
	}
}

func imageListCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	err = p.ListImages(ctx, "")
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

	c := api.NewConfig()

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
	images, err := p.GetImages(ctx, "")
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
	cmdImageSync.PersistentFlags().StringVarP(&sourceCloud, "source-cloud", "s", onprem.ProviderName, "cloud platform [gcp, aws, do, vultr, onprem, hyper-v, upcloud]")
	return cmdImageSync
}

func imageSyncCommandHandler(cmd *cobra.Command, args []string) {
	image := args[0]
	// TODO only accepts onprem for now, implement for other source providers later
	source, _ := cmd.Flags().GetString("source-cloud")
	if source != onprem.ProviderName {
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

func imageCatCommand() *cobra.Command {
	var cmdCat = &cobra.Command{
		Use:   "cat <image_name> <src>",
		Short: "cat from image to local filesystem",
		Run:   imageCatCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdCat
}

func imageCatCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalImageReader(cmd.Flags(), args)
	defer reader.Close()
	imageCat(cmd, args, reader)
}

func imageCat(cmd *cobra.Command, args []string, reader *fs.Reader) {
	for _, srcPath := range args[1:] {
		_, err := reader.Stat(srcPath)
		if err != nil {
			log.Errorf("Invalid source '%s': %v", srcPath, err)
			continue
		}

		ir, err := reader.ReadFile(srcPath)
		if err != nil {
			fmt.Println(err)
		}

		b, err := io.ReadAll(ir)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(string(b))
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
	flags.BoolP("bootfs", "", false, "use boot filesystem")
	return cmdCopy
}

func imageCopyCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalImageReader(cmd.Flags(), args)
	defer reader.Close()
	imageCopy(cmd, args, reader)
}

func imageCopy(cmd *cobra.Command, args []string, reader *fs.Reader) {
	flags := cmd.Flags()
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
				err = fmt.Errorf("cannot copy '%s' to '%s': %v", srcPath, dest, err)
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
		return fmt.Errorf("cannot read directory '%s': %v", src, err)
	}
	if _, err = os.Stat(dest); os.IsNotExist(err) {
		if err = os.Mkdir(dest, 0755); err != nil {
			return fmt.Errorf("cannot create directory '%s': %v", src, err)
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
				err = fmt.Errorf("cannot copy '%s' to '%s': %v", srcPath, destPath, err)
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
		Use:   "ls <image_name> [<path>]",
		Short: "list files and directories in image",
		Run:   imageLsCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmdLs.PersistentFlags()
	flags.BoolP("long-format", "l", false, "use a long listing format")
	flags.BoolP("bootfs", "", false, "use boot filesystem")
	return cmdLs
}

func imageLsCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalImageReader(cmd.Flags(), args)
	defer reader.Close()
	imageLs(cmd, args, reader)
}

func imageLs(cmd *cobra.Command, args []string, reader *fs.Reader) {
	var srcPath string
	if len(args) < 2 {
		srcPath = "/"
	} else if len(args) > 2 {
		exitForCmd(cmd, "Too many arguments")
	} else {
		srcPath = args[1]
	}
	longFormat, _ := cmd.Flags().GetBool("long-format")
	fileInfo, err := reader.Stat(srcPath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot access '%s': %v", srcPath, err))
	}
	switch fileInfo.Mode() {
	case os.ModeDir:
		dirEntries, err := reader.ReadDir(srcPath)
		if err != nil {
			exitWithError(fmt.Sprintf("cannot read directory '%s': %v", srcPath, err))
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
	var cmdTree = &cobra.Command{
		Use:   "tree <image_name>",
		Short: "display image filesystem contents in tree format",
		Run:   imageTreeCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmdTree.PersistentFlags()
	flags.BoolP("bootfs", "", false, "use boot filesystem")
	return cmdTree
}

func imageTreeCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	reader := getLocalImageReader(flags, args)

	jsonOutput, err := flags.GetBool("json")
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	if jsonOutput {
		files, err := dumpFSEntryJSON(reader, "/")
		if err != nil {
			fmt.Println(err)
		}
		reader.Close()

		json.NewEncoder(os.Stdout).Encode(files)
		return
	}

	dumpFSEntry(reader, "/", 0)
	reader.Close()
}

func dumpFSEntryJSON(reader *fs.Reader, srcPath string) ([]string, error) {
	files := []string{}

	fileInfo, err := reader.Stat(srcPath)

	if err != nil {
		exitWithError(fmt.Sprintf("cannot access '%s': %v", srcPath, err))
	}

	switch fileInfo.Mode() {
	case os.ModeDir:
		dirEntries, err := reader.ReadDir(srcPath)
		if err != nil {
			exitWithError(fmt.Sprintf("cannot read directory '%s': %v", srcPath, err))
		}
		for _, entry := range dirEntries {
			p, err := dumpFSEntryJSON(reader, path.Join(srcPath, entry.Name()))
			if err != nil {
				fmt.Println(err)
			}
			for x := 0; x < len(p); x++ {
				if fileInfo.Name() == "/" {
					files = append(files, fileInfo.Name()+p[x])
				} else {
					files = append(files, fileInfo.Name()+"/"+p[x])
				}
			}
		}
	case os.ModeSymlink:
		files = append(files, fileInfo.Name())
	case 0:
		files = append(files, fileInfo.Name())
	}
	return files, nil
}

func dumpFSEntry(reader *fs.Reader, srcPath string, indent int) {
	fileInfo, err := reader.Stat(srcPath)
	if err != nil {
		exitWithError(fmt.Sprintf("cannot access '%s': %v", srcPath, err))
	}
	switch fileInfo.Mode() {
	case os.ModeDir:
		fmt.Println(getDumpLine(indent, fileInfo, log.ConsoleColors.Blue()))
		dirEntries, err := reader.ReadDir(srcPath)
		if err != nil {
			exitWithError(fmt.Sprintf("cannot read directory '%s': %v", srcPath, err))
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

func imageEnvCommand() *cobra.Command {
	var cmdEnv = &cobra.Command{
		Use:   "env <image_name>",
		Short: "list environment variables in image",
		Run:   imageEnvCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdEnv
}

func imageEnvCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalImageReader(cmd.Flags(), args)
	envVars := reader.ListEnv()
	reader.Close()
	if len(envVars) == 0 {
		fmt.Println("(none)")
	} else {
		for name, value := range envVars {
			fmt.Printf("%s: %s\n", name, value)
		}
	}
}

func getLocalImageReader(flags *pflag.FlagSet, args []string) *fs.Reader {
	c := api.NewConfig()
	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}
	if c.CloudConfig.Platform != onprem.ProviderName {
		exitWithError("Image subcommand not implemented yet for cloud images")
	}
	imageName := args[0]
	imagePath := path.Join(api.LocalImageDir, imageName)
	if _, err := os.Stat(imagePath); err != nil {
		if err != nil {
			if os.IsNotExist(err) {
				exitWithError(fmt.Sprintf("Local image %s not found", imageName))
			} else {
				exitWithError(fmt.Sprintf("Cannot read image %s: %v", imageName, err))
			}
		}
	}

	var reader *fs.Reader
	bootFS, _ := flags.GetBool("bootfs")

	switch {
	case bootFS:
		reader, err = fs.NewReaderBootFS(imagePath)
	default:
		reader, err = fs.NewReader(imagePath)
	}
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot load image %s: %v", imageName, err))
	}
	return reader
}

func imageMirrorCommand() *cobra.Command {

	var cmdMirror = &cobra.Command{
		Use:   "mirror <image_name> <src_region> <dst_region>",
		Short: "copies an image from one region to another",
		Run:   imageMirrorHandler,
		Args:  cobra.ExactArgs(3),
	}
	return cmdMirror

}

func imageMirrorHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	// the first argument for the command can be considered as the zone
	c.CloudConfig.Zone = args[1]

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	mirrorer, ok := p.(api.Mirrorer)
	if !ok {
		exitWithError(fmt.Sprintf("mirroring images for cloud provider %s is not yet implemented by ops", c.CloudConfig.Platform))
	}

	newImageID, err := mirrorer.MirrorImage(ctx, args[0], args[1], args[2])
	if err != nil {
		exitWithError(err.Error())
	}
	fmt.Println("Image was successfully mirrored. New image id -", newImageID)
}
