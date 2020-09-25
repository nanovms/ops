package cmd

import (
	"log"
	"path"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func volumeCreateCommandHandler(cmd *cobra.Command, args []string) {
	name := args[0]
	if name == "" {
		log.Fatal("name required")
	}
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

	var vol api.VolumeService
	if provider == "" {
		vol = api.NewLocalVolume()
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err = vol.CreateVolume(name, data, size, conf)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeCreateCommand() *cobra.Command {
	var data string
	var size string
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

func volumeListCommandHandler(cmd *cobra.Command, args []string) {
	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	provider, _ := cmd.Flags().GetString("target-cloud")

	var err error
	var vol api.VolumeService
	if provider == "" {
		vol = api.NewLocalVolume()
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = vol.GetAllVolume(conf)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeListCommand() *cobra.Command {
	cmdVolumeList := &cobra.Command{
		Use:   "list",
		Short: "list volume",
		Run:   volumeListCommandHandler,
	}
	return cmdVolumeList
}

func volumeDeleteCommandHandler(cmd *cobra.Command, args []string) {
	id := args[0]
	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	provider, _ := cmd.Flags().GetString("target-cloud")

	var err error
	var vol api.VolumeService
	if provider == "" {
		vol = api.NewLocalVolume()
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = vol.DeleteVolume(id, conf)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeDeleteCommand() *cobra.Command {
	cmdVolumeDelete := &cobra.Command{
		Use:   "delete <volume_id>",
		Short: "delete volume",
		Run:   volumeDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}
	return cmdVolumeDelete
}

func volumeAttachCommandHandler(cmd *cobra.Command, args []string) {
	volume := args[0]
	image := args[1]
	mount := args[2]
	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	provider, _ := cmd.Flags().GetString("target-cloud")

	var err error
	var vol api.VolumeService
	if provider == "" {
		log.Fatal("please select on of the cloud platform in config [onprem, aws, gcp, do, vsphere, vultr]")
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = vol.AttachVolume(image, volume, mount, conf)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeAttachCommand() *cobra.Command {
	cmdVolumeAttach := &cobra.Command{
		Use:   "attach <volume_name> <image_name> <mount_path>",
		Short: "attach volume from image",
		Run:   volumeAttachCommandHandler,
		Args:  cobra.MinimumNArgs(3),
	}
	return cmdVolumeAttach
}

func volumeDetachCommandHandler(cmd *cobra.Command, args []string) {
	volume := args[0]
	image := args[1]
	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	provider, _ := cmd.Flags().GetString("target-cloud")

	var err error
	var vol api.VolumeService
	if provider == "" {
		log.Fatal("please select on of the cloud platform in config [onprem, aws, gcp, do, vsphere, vultr]")
	} else {
		vol, err = getCloudProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = vol.DetachVolume(image, volume, conf)
	if err != nil {
		log.Fatal(err)
	}
}

func volumeDetachCommand() *cobra.Command {
	cmdVolumeDetach := &cobra.Command{
		Use:   "detach volume <volume_name> <image_name>",
		Short: "detach volume from image",
		Run:   volumeDetachCommandHandler,
		Args:  cobra.MinimumNArgs(2),
	}
	return cmdVolumeDetach
}

// VolumeCommands handles volumes related operations
func VolumeCommands() *cobra.Command {
	var config string
	var provider string
	var nightly bool
	cmdVolume := &cobra.Command{
		Use:       "volume",
		Short:     "manage nanos volumes",
		ValidArgs: []string{"create, list, delete, attach, detach"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdVolume.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdVolume.PersistentFlags().StringVarP(&provider, "target-cloud", "t", "", "cloud platform [onprem, aws, gcp, do, vsphere, vultr]")
	cmdVolume.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	// TODO cloud related flags (use config for now)
	cmdVolume.AddCommand(volumeCreateCommand())
	cmdVolume.AddCommand(volumeListCommand())
	cmdVolume.AddCommand(volumeDeleteCommand())
	cmdVolume.AddCommand(volumeAttachCommand())
	cmdVolume.AddCommand(volumeDetachCommand())
	return cmdVolume
}
