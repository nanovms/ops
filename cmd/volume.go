package cmd

import (
	"log"

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
	provider, _ := cmd.Flags().GetString("provider")
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
	fixupConfigImages(conf, version)

	vol := api.NewVolume(conf)
	err = vol.Create(name, data, size, provider, conf.Mkfs)
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
	vol := api.NewVolume(conf)
	err := vol.GetAll()
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
	name := args[0]
	config, _ := cmd.Flags().GetString("config")
	conf := unWarpConfig(config)
	vol := api.NewVolume(conf)
	err := vol.Delete(name)
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

// VolumeCommands handles volumes related operations
func VolumeCommands() *cobra.Command {
	var config string
	var provider string
	var nightly bool
	cmdVolume := &cobra.Command{
		Use:       "volume",
		Short:     "manage nanos volumes",
		ValidArgs: []string{"create, list"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdVolume.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdVolume.PersistentFlags().StringVarP(&provider, "provider", "p", "", "cloud provider")
	cmdVolume.PersistentFlags().BoolVarP(&nightly, "nightly", "n", false, "nightly build")
	cmdVolume.AddCommand(volumeCreateCommand())
	cmdVolume.AddCommand(volumeListCommand())
	cmdVolume.AddCommand(volumeDeleteCommand())
	return cmdVolume
}
