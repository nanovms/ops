package cmd

import (
	"fmt"
	"os"
	"runtime"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func updateCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("Checking for updates...")
	err := api.DoUpdate(fmt.Sprintf(api.OpsReleaseUrl, runtime.GOOS))
	if err != nil {
		fmt.Println("Failed to update.", err)
	} else {
		fmt.Println("Successfully updated ops. Please restart.")
	}
	os.Exit(0)
}

func UpdateCommand() *cobra.Command {
	var cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "check for updates",
		Run:   updateCommandHandler,
	}
	return cmdUpdate
}
