package cmd

import (
	"log"
	"path"
	"strconv"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func volumeCreateCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	data, _ := cmd.Flags().GetString("data")
	size, _ := cmd.Flags().GetString("size")
	config, _ := cmd.Flags().GetString("config")
	provider, _ := cmd.Flags().GetString("target-cloud")
	nightly, _ := cmd.Flags().GetBool("nightly")

	conf := unWarpConfig(config)
	conf.NightlyBuild = nightly
	var err error
	var version string
	if conf.NightlyBuild {
		version, err = downloadNightlyImages(conf)
	} else {
		version, err = downloadReleaseImages()
	}
	if err != nil {
		log.Fatal(err)
	}
	if conf.Mkfs == "" {
		conf.Mkfs = path.Join(api.GetOpsHome(), version, "mkfs")
	}
	conf.BuildDir = api.LocalVolumeDir

	var vol api.VolumeService
	if provider == "onprem" {
		vol = &api.OnPrem{}
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	res, err := vol.CreateVolume(conf, name, data, size, provider)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("volume: %s created with UUID %s and label %s\n", res.Name, res.ID, res.Label)
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
	cmdVolumeCreate.PersistentFlags().StringVarP(&size, "size", "s", strconv.Itoa(api.MinimumVolumeSize), "volume initial size")
	return cmdVolumeCreate
}

func volumeListCommandHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	provider, _ := cmd.Flags().GetString("target-cloud")
	conf := unWarpConfig(config)

	var vol api.VolumeService
	var err error
	if provider == "onprem" {
		vol = &api.OnPrem{}
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}

	conf.BuildDir = api.LocalVolumeDir

	err = vol.GetAllVolumes(conf)
	if err != nil {
		log.Fatal(err)
	}
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

func volumeDeleteCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	config, _ := cmd.Flags().GetString("config")
	provider, _ := cmd.Flags().GetString("target-cloud")
	conf := unWarpConfig(config)

	var vol api.VolumeService
	var err error
	if provider == "onprem" {
		vol = &api.OnPrem{}
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}

	conf.BuildDir = api.LocalVolumeDir

	err = vol.DeleteVolume(conf, name)
	if err != nil {
		log.Fatal(err)
	}
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

func volumeAttachCommandHandler(cmd *cobra.Command, args []string) {
	image := args[0]
	name := args[1]
	mount := args[2]
	config, _ := cmd.Flags().GetString("config")
	provider, _ := cmd.Flags().GetString("provider")
	conf := unWarpConfig(config)

	var vol api.VolumeService
	var err error
	if provider == "onprem" {
		vol = &api.OnPrem{}
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = vol.AttachVolume(conf, image, name, mount)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeAttachCommand() *cobra.Command {
	cmdVolumeAttach := &cobra.Command{
		Use:   "attach <image_name> <volume_name> <mount_path>",
		Short: "attach volume",
		Run:   volumeAttachCommandHandler,
		Args:  cobra.MinimumNArgs(3),
	}
	return cmdVolumeAttach
}

// VolumeCommands handles volumes related operations
func VolumeCommands() *cobra.Command {
	var config, provider string
	var nightly bool
	cmdVolume := &cobra.Command{
		Use:       "volume",
		Short:     "manage nanos volumes",
		ValidArgs: []string{"create, list, delete, attach"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdVolume.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdVolume.PersistentFlags().StringVarP(&provider, "target-cloud", "t", "onprem", "cloud provider")
	cmdVolume.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdVolume.AddCommand(volumeCreateCommand())
	cmdVolume.AddCommand(volumeListCommand())
	cmdVolume.AddCommand(volumeDeleteCommand())
	cmdVolume.AddCommand(volumeAttachCommand())
	return cmdVolume
}
