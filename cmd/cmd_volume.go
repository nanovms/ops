package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/types"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

// VolumeCommands handles volumes related operations
func VolumeCommands() *cobra.Command {
	cmdVolume := &cobra.Command{
		Use:       "volume",
		Short:     "manage nanos volumes",
		ValidArgs: []string{"create, list, delete, attach, tree, ls, cp"},
		Args:      cobra.OnlyValidArgs,
	}

	persistentFlags := cmdVolume.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistProviderCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	cmdVolume.AddCommand(volumeCreateCommand())
	cmdVolume.AddCommand(volumeListCommand())
	cmdVolume.AddCommand(volumeDeleteCommand())
	cmdVolume.AddCommand(volumeAttachCommand())
	cmdVolume.AddCommand(volumeDetachCommand())
	cmdVolume.AddCommand(volumeTreeCommand())
	cmdVolume.AddCommand(volumeLsCommand())
	cmdVolume.AddCommand(volumeCopyCommand())
	cmdVolume.AddCommand(volumeInfoCommand())
	return cmdVolume
}

func volumeCreateCommand() *cobra.Command {
	var data, size, typeof, iops, throughput string
	cmdVolumeCreate := &cobra.Command{
		Use:   "create <volume_name>",
		Short: "create volume",
		Run:   volumeCreateCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdVolumeCreate.PersistentFlags().StringVarP(&data, "data", "d", "", "volume data source")
	cmdVolumeCreate.PersistentFlags().StringVarP(&size, "size", "s", "", "volume initial size")
	cmdVolumeCreate.PersistentFlags().StringVarP(&typeof, "typeof", "", "", "volume type")
	cmdVolumeCreate.PersistentFlags().StringVarP(&iops, "iops", "", "", "volume iops")
	cmdVolumeCreate.PersistentFlags().StringVarP(&throughput, "throughput", "", "", "volume throughput")

	return cmdVolumeCreate
}

func volumeCreateCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	data, _ := cmd.Flags().GetString("data")
	size, _ := cmd.Flags().GetString("size")
	typeof, _ := cmd.Flags().GetString("typeof")
	iops, _ := cmd.Flags().GetString("iops")
	throughput, _ := cmd.Flags().GetString("throughput")

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}
	if size != "" {
		c.BaseVolumeSz = size
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	cv := types.CloudVolume{
		Name:   name,
		Typeof: typeof,
	}

	if iops != "" {
		cv.Iops, err = strconv.ParseInt(iops, 10, 64)
		if err != nil {
			fmt.Println(err)
		}
	}

	if throughput != "" {
		cv.Throughput, err = strconv.ParseInt(throughput, 10, 64)
		if err != nil {
			fmt.Println(err)
		}
	}

	res, err := p.CreateVolume(ctx, cv, data, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}
	log.Infof("volume: %s created with UUID %s and label %s\n", res.Name, res.ID, res.Label)
}

// TODO might be nice to be able to filter by name/label
// api.GetVolumes can be implemented to achieve this
func volumeListCommand() *cobra.Command {
	cmdVolumeList := &cobra.Command{
		Use:   "list",
		Short: "list volume",
		Run:   volumeListCommandHandler,
	}
	return cmdVolumeList
}

func volumeListCommandHandler(cmd *cobra.Command, args []string) {
	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	volumes, err := p.GetAllVolumes(ctx)
	if err != nil {
		log.Fatal(err)
	}

	api.PrintVolumesList(volumes)
}

func volumeDeleteCommand() *cobra.Command {
	cmdVolumeDelete := &cobra.Command{
		Use:   "delete <volume_name:volume_uuid>",
		Short: "delete volume",
		Run:   volumeDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdVolumeDelete
}

func volumeDeleteCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	err = p.DeleteVolume(ctx, name)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeAttachCommand() *cobra.Command {
	cmdVolumeAttach := &cobra.Command{
		Use:   "attach <instance_name> <volume_name>",
		Short: "attach volume",
		Run:   volumeAttachCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdVolumeAttach
}

func volumeAttachCommandHandler(cmd *cobra.Command, args []string) {
	instanceName := args[0]
	volumeName := args[1]
	attachID := -1
	attachIDPrefix := "%"
	if strings.HasPrefix(volumeName, attachIDPrefix) {
		attachStrings := strings.Split(strings.TrimPrefix(volumeName, attachIDPrefix), ":")
		if len(attachStrings) == 2 {
			if id, err := strconv.Atoi(attachStrings[0]); err == nil {
				attachID = id
				volumeName = attachStrings[1]
			}
		}
	}

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	err = p.AttachVolume(ctx, instanceName, volumeName, attachID)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeDetachCommand() *cobra.Command {
	cmdVolumeDetach := &cobra.Command{
		Use:   "detach <instance_name> <volume_name>",
		Short: "detach volume",
		Run:   volumeDetachCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdVolumeDetach
}

func volumeDetachCommandHandler(cmd *cobra.Command, args []string) {
	instance := args[0]
	name := args[1]

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	err = p.DetachVolume(ctx, instance, name)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeTreeCommand() *cobra.Command {
	var cmdTree = &cobra.Command{
		Use:   "tree <volume_name:volume_uuid>",
		Short: "display volume filesystem contents in tree format",
		Run:   volumeTreeCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdTree
}

func volumeTreeCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalVolumeReader(cmd, args)
	dumpFSEntry(reader, "/", 0)
	reader.Close()
}

func volumeLsCommand() *cobra.Command {
	var cmdLs = &cobra.Command{
		Use:   "ls <volume_name:volume_uuid> [<path>]",
		Short: "list files and directories in volume",
		Run:   volumeLsCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	flags := cmdLs.PersistentFlags()
	flags.BoolP("long-format", "l", false, "use a long listing format")
	return cmdLs
}

func volumeLsCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalVolumeReader(cmd, args)
	defer reader.Close()
	imageLs(cmd, args, reader)
}

func volumeCopyCommand() *cobra.Command {
	var cmdCopy = &cobra.Command{
		Use:   "cp <volume_name:volume_uuid> <src>... <dest>",
		Short: "copy files from volume to local filesystem",
		Run:   volumeCopyCommandHandler,
		Args:  cobra.MinimumNArgs(3),
	}
	flags := cmdCopy.PersistentFlags()
	flags.BoolP("recursive", "r", false, "copy directories recursively")
	flags.BoolP("dereference", "L", false, "always follow symbolic links in volume")
	return cmdCopy
}

func volumeCopyCommandHandler(cmd *cobra.Command, args []string) {
	reader := getLocalVolumeReader(cmd, args)
	defer reader.Close()
	imageCopy(cmd, args, reader)
}

func volumeInfoCommand() *cobra.Command {
	var cmdInfo = &cobra.Command{
		Use:   "info <volume_file_path>",
		Short: "get label/uuid of file system",
		Run:   volumeInfoCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdInfo
}

func volumeInfoCommandHandler(cmd *cobra.Command, args []string) {
	filePath := args[0]
	reader := getLocalReaderFromFile(cmd, filePath)
	defer reader.Close()
	label := reader.GetLabel()
	uuid := reader.GetUUID()
	json, _ := cmd.Flags().GetBool("json")
	if json {
		fmt.Printf("{\"label\":\"%s\",\"uuid\":\"%s\"}\n", label, uuid)
	} else {
		fmt.Printf("%s:%s\n", label, uuid)
	}
}

func getLocalReaderFromFile(cmd *cobra.Command, filePath string) *fs.Reader {
	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}
	if c.CloudConfig.Platform != onprem.ProviderName {
		exitWithError("Volume subcommand not implemented for cloud volumes")
	}
	if _, err := os.Stat(filePath); err != nil {
		if err != nil {
			if os.IsNotExist(err) {
				exitWithError(fmt.Sprintf("Local file %s not found", filePath))
			} else {
				exitWithError(fmt.Sprintf("Cannot read file %s: %v", filePath, err))
			}
		}
	}
	reader, err := fs.NewReader(filePath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot load information from %s: %v", filePath, err))
	}
	return reader
}

func getLocalVolumeReader(cmd *cobra.Command, args []string) *fs.Reader {
	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}
	if c.CloudConfig.Platform != onprem.ProviderName {
		exitWithError("Volume subcommand not implemented yet for cloud volumes")
	}
	_, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}
	volumeNameID := args[0]
	query := map[string]string{
		"label": volumeNameID,
		"id":    volumeNameID,
	}
	localVolumeDir := ctx.Config().VolumesDir
	volumes, err := onprem.GetVolumes(localVolumeDir, query)

	if len(volumes) > 1 {
		exitWithError(fmt.Sprintf("Found %d volumes with the same label %s, please select by uuid", len(volumes), volumeNameID))
	}
	volumePath := path.Join(volumes[0].Path)
	if _, err := os.Stat(volumePath); err != nil {
		if err != nil {
			if os.IsNotExist(err) {
				exitWithError(fmt.Sprintf("Local volume %s not found", volumeNameID))
			} else {
				exitWithError(fmt.Sprintf("Cannot read volume %s: %v", volumeNameID, err))
			}
		}
	}
	reader, err := fs.NewReader(volumePath)
	if err != nil {
		exitWithError(fmt.Sprintf("Cannot load volume %s: %v", volumeNameID, err))
	}
	return reader
}

func getVolumeCommandDefaultConfig(cmd *cobra.Command) (c *types.Config, err error) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c = api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, providerFlags)
	err = mergeContainer.Merge(c)

	return
}
