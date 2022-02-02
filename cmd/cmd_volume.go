package cmd

import (
	"log"

	"github.com/nanovms/ops/lepton"
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
	return cmdVolume
}

func volumeCreateCommand() *cobra.Command {
	var data, size string
	cmdVolumeCreate := &cobra.Command{
		Use:   "create <volume_name>",
		Short: "create volume",
		Run:   volumeCreateCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	cmdVolumeCreate.PersistentFlags().StringVarP(&data, "data", "d", "", "volume data source")
	cmdVolumeCreate.PersistentFlags().StringVarP(&size, "size", "s", "", "volume initial size")
	return cmdVolumeCreate
}

func volumeCreateCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	data, _ := cmd.Flags().GetString("data")
	size, _ := cmd.Flags().GetString("size")

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

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	err = p.AttachVolume(ctx, instanceName, volumeName)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeDetachCommand() *cobra.Command {
	cmdVolumeDetach := &cobra.Command{
		Use:   "detach <image_name> <volume_name>",
		Short: "detach volume",
		Run:   volumeDetachCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdVolumeDetach
}

func volumeDetachCommandHandler(cmd *cobra.Command, args []string) {
	image := args[0]
	name := args[1]

	c, err := getVolumeCommandDefaultConfig(cmd)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		log.Fatal(err)
	}

	err = p.DetachVolume(ctx, image, name)
	if err != nil {
		log.Fatal(err)
	}
}

func getVolumeCommandDefaultConfig(cmd *cobra.Command) (c *types.Config, err error) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c = lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, providerFlags)
	err = mergeContainer.Merge(c)

	return
}
