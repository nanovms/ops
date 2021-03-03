package cmd

import (
	"fmt"
	"os"
	"runtime"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

// UpdateCommand provides update related commands
func UpdateCommand() *cobra.Command {
	var cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "check for updates",
		Run:   updateCommandHandler,
	}
	return cmdUpdate
}

func updateCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("Checking for updates...")
	err := api.DoUpdate(fmt.Sprintf(api.OpsReleaseURL, runtime.GOOS))
	if err != nil {
		fmt.Println("Failed to update.", err)
	} else {
		fmt.Println("Updates ops to latest release.")
	}
	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	if local == "0.0" || parseVersion(local, 4) != parseVersion(remote, 4) {
		err = api.DownloadReleaseImages(remote)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = api.DownloadCommonFiles()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		api.UpdateLocalRelease(remote)
		fmt.Printf("Update nanos to %s version.\n", remote)
	}
	os.Exit(0)
}
