package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/nanovms/ops/fs"
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
		ValidArgs: []string{"create, list, delete, attach"},
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
	return cmdVolume
}

func volumeCreateCommand() *cobra.Command {
	var (
		data           string
		size           string
		sourceImage    string
		sourceVolume   string
		sourceSnapshot string
	)
	cmdVolumeCreate := &cobra.Command{
		Use:   "create <volume_name>",
		Short: "create volume",
		Run:   volumeCreateCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdVolumeCreate.PersistentFlags().StringVarP(&data, "data", "d", "", "volume local data source, ignored when cloning from an existing source (image,volume,snapshot)")
	cmdVolumeCreate.PersistentFlags().StringVarP(&size, "size", "s", "", "volume initial size")
	cmdVolumeCreate.PersistentFlags().StringVarP(&sourceVolume, "volume", "", "", "name of an existing volume to clone from")
	cmdVolumeCreate.PersistentFlags().StringVarP(&sourceSnapshot, "snapshot", "", "", "name of an existing snapshot to clone from")
	cmdVolumeCreate.PersistentFlags().StringVarP(&sourceImage, "image", "", "", "name of an existing image to clone from")
	cmdVolumeCreate.PersistentFlags().Lookup("image").NoOptDefVal = "\\"
	cmdVolumeCreate.MarkFlagsMutuallyExclusive("image", "volume", "snapshot")
	cmdVolumeCreate.MarkFlagsMutuallyExclusive("data", "volume")
	cmdVolumeCreate.MarkFlagsMutuallyExclusive("data", "snapshot")

	return cmdVolumeCreate
}

func volumeCreateCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	data, _ := cmd.Flags().GetString("data")
	size, _ := cmd.Flags().GetString("size")
	sourceImage, _ := cmd.Flags().GetString("image")
	sourceVolume, _ := cmd.Flags().GetString("volume")
	sourceSnapshot, _ := cmd.Flags().GetString("snapshot")

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

	if sourceImage == cmd.Flag("image").NoOptDefVal {
		res, err := p.CreateVolumeImage(ctx, name, data, c.CloudConfig.Platform)
		if err != nil {
			exitWithError(err.Error())
		}
		log.Printf("volume: created image %s with UUID %s and label %s\n", res.Name, res.ID, res.Label)
		return
	}

	if sourceImage != "" {
		if err = p.CreateVolumeFromSource(ctx, "image", sourceImage, name, c.CloudConfig.Platform); err != nil {
			exitWithError(err.Error())
		}
		log.Printf("volume: created disk with label %s from image %s\n", name, sourceImage)
		return
	}
	if sourceVolume != "" {
		if err = p.CreateVolumeFromSource(ctx, "disk", sourceVolume, name, c.CloudConfig.Platform); err != nil {
			exitWithError(err.Error())
		}
		log.Printf("volume: created disk with label %s from volume %s\n", name, sourceVolume)
		return
	}
	if sourceSnapshot != "" {
		if err = p.CreateVolumeFromSource(ctx, "snapshot", sourceSnapshot, name, c.CloudConfig.Platform); err != nil {
			exitWithError(err.Error())
		}
		log.Printf("volume: created disk with label %s from snapshot %s\n", name, sourceSnapshot)
		return
	}

	res, err := p.CreateVolume(ctx, name, data, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}
	log.Printf("volume: %s created with UUID %s and label %s\n", res.Name, res.ID, res.Label)
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

func getLocalVolumeReader(cmd *cobra.Command, args []string) *fs.Reader {
	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}
	if c.CloudConfig.Platform != onprem.ProviderName {
		exitWithError("Volume subcommand not implemented yet for cloud images")
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
