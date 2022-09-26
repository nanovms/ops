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

	cmdUpdate.PersistentFlags().BoolP("arm", "", false, "grab latest arm release")

	return cmdUpdate
}

func updateCommandHandler(cmd *cobra.Command, args []string) {

	// this flag is to see if user wants arm release
	// important to note that it could be used on x86 || arm64
	// empty string means it defaults to x86
	arm, _ := cmd.Flags().GetBool("arm")
	arch := ""
	if arm {
		arch = "arm"
	}

	log.Info("Checking for updates...")

	// this check is to see if ops itself is running on arm
	platform := runtime.GOOS
	if runtime.GOARCH == "arm64" {
		platform += "/aarch64"
	}

	err := api.DoUpdate(fmt.Sprintf(api.OpsReleaseURL, platform))
	if err != nil {
		log.Errorf("Failed to update. %v", err)
	} else {
		log.Info("Updates ops to latest release.")
	}

	local, remote := api.LocalReleaseVersion, api.LatestReleaseVersion
	_, err = os.Stat(path.Join(api.GetOpsHome(), local))

	// FIXME: for now if arch is set we'll dl - should fix this in the future
	// and look for the arch first
	if local == "0.0" || (arch != "") || parseVersion(local, 4) != parseVersion(remote, 4) || os.IsNotExist(err) {
		err = api.DownloadReleaseImages(remote, arch)
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
