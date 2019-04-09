package cmd

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

func runningAsRoot() bool {
	cmd := exec.Command("id", "-u")
	output, _ := cmd.Output()
	i, _ := strconv.Atoi(string(output[:len(output)-1]))
	return i == 0
}

func netSetupCommandHandler(cmd *cobra.Command, args []string) {
	if !runningAsRoot() {
		fmt.Println("net command needs root permission")
		return
	}
	if err := setupBridgeNetwork(); err != nil {
		panic(err)
	}
}

func netResetCommandHandler(cmd *cobra.Command, args []string) {
	if !runningAsRoot() {
		fmt.Println("net command needs root permission")
		return
	}
	if err := resetBridgeNetwork(); err != nil {
		panic(err)
	}
}

// NetCommands provides net related commands
func NetCommands() *cobra.Command {
	var cmdNetSetup = &cobra.Command{
		Use:   "setup",
		Short: "Setup bridged network",
		Run:   netSetupCommandHandler,
	}

	var cmdNetReset = &cobra.Command{
		Use:   "reset",
		Short: "Reset bridged network",
		Run:   netResetCommandHandler,
	}

	var cmdNet = &cobra.Command{
		Use:       "net",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"setup", "reset"},
		Short:     "Configure bridge network",
	}

	cmdNet.AddCommand(cmdNetReset)
	cmdNet.AddCommand(cmdNetSetup)
	return cmdNet
}
