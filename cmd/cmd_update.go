package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
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
	log.Info("Checking for updates...")
	err := api.DoUpdate(fmt.Sprintf(api.OpsReleaseURL, runtime.GOOS))
	if err != nil {
		log.Errorf("Failed to update. %v", err)
	} else {
		log.Info("Updates ops to latest release.")
	}
	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	_, err = os.Stat(path.Join(api.GetOpsHome(), local))
	if local == "0.0" || parseVersion(local, 4) != parseVersion(remote, 4) || os.IsNotExist(err) {
		err = api.DownloadReleaseImages(remote)
		if err != nil {
			log.Fatal(err)
		}
		err = api.DownloadCommonFiles()
		if err != nil {
			log.Fatal(err)
		}
		api.UpdateLocalRelease(remote)
		fmt.Printf("Update nanos to %s version.\n", remote)
	}
	os.Exit(0)
}
