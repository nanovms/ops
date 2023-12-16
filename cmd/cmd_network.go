// currently for linux only and relies on bridgetools and dnsmasq

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// NetworkCommands have the capability of creating/destroying networks.
// linux only for now - mac uses vmnet-bridged.
func NetworkCommands() *cobra.Command {
	var cmdNetwork = &cobra.Command{
		Use:       "network",
		Short:     "manage nanos images",
		ValidArgs: []string{"create", "list", "delete"},
		Args:      cobra.OnlyValidArgs,
	}

	cmdNetwork.AddCommand(networkCreateCommand())
	cmdNetwork.AddCommand(networkListCommand())
	cmdNetwork.AddCommand(networkDeleteCommand())

	return cmdNetwork
}

func networkCreateCommand() *cobra.Command {

	var cmdNetworkCreate = &cobra.Command{
		Use:   "create network",
		Short: "create network",
		Run:   networkCreateCommandHandler,
	}

	return cmdNetworkCreate
}

func networkCreateCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("stubbed")

	// mv me elsewhere
	// option
	bridge := "br0"

	// option; also break out class-c to provide range
	network := "192.168.1.1/24"

	ecmd := exec.Command("sudo", "brctl", "show", bridge)
	out, err := ecmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}

	if strings.Contains(string(out), "does not exist!") {

		ecmd = exec.Command("sudo", "brctl", "addbr", bridge)
		_, err = ecmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		ecmd = exec.Command("sudo", "ifconfig", bridge, "inet", network)
		_, err = ecmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		ecmd = exec.Command("sudo", "dnsmasq", "--bind-interfaces", "--interface="+bridge, "--except-interface=lo", "--leasefile-ro", "--dhcp-range=192.168.1.2,192.168.1.251,12")
		out, err = ecmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(string(out))
	}

	os.Exit(1)
}

func networkListCommand() *cobra.Command {
	var cmdNetworkList = &cobra.Command{
		Use:   "list",
		Short: "list networks",
		Run:   networkListCommandHandler,
	}
	return cmdNetworkList
}

func networkListCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("stubbed")
	os.Exit(1)
}

func networkDeleteCommand() *cobra.Command {
	var cmdNetworkDelete = &cobra.Command{
		Use:   "delete <network_name>",
		Short: "delete network",
		Run:   networkDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdNetworkDelete
}

func networkDeleteCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("stubbed")

	killBridge("br0")

	os.Exit(1)
}

// mv elsewhere and get rid of shelling

func killBridge(bridgeName string) {
	log := false

	fmt.Printf("killing bridge %s\n", bridgeName)
	ecmd := exec.Command("sudo", "ifconfig", bridgeName, "down")
	out, err := ecmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}

	if log {
		fmt.Println(string(out))
	}

	pid := getPidOfBridge(bridgeName)

	if log {
		fmt.Printf("killing dnsmasq - has a pid of #%s#\n", pid)
	}

	ecmd = exec.Command("sudo", "kill", "-9", pid)
	out, err = ecmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}

	if log {
		fmt.Println(string(out))
	}

	ecmd = exec.Command("sudo", "brctl", "delbr", bridgeName)
	out, err = ecmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}

	if log {
		fmt.Println(string(out))
	}
}

func execCmd(cmdStr string) (output string, err error) {
	ecmd := exec.Command("/bin/bash", "-c", cmdStr)
	out, err := ecmd.CombinedOutput()
	if err != nil {
		return
	}
	output = string(out)
	return
}

func getPidOfBridge(bridgeName string) string {
	pid, err := execCmd("ps aux | grep dnsmasq | grep " + bridgeName + " | awk {'print $2'}")
	if err != nil {
		fmt.Println(err)
	}

	if strings.Contains(pid, "\n") {
		rpid := strings.Split(pid, "\n")
		pid = strings.TrimSpace(rpid[0])
	}

	fmt.Printf("found pid of %s\n", pid)
	return pid
}

func emptyBridge(bridgeName string) bool {
	o, err := execCmd("brctl show | grep " + bridgeName)
	if err != nil {
		fmt.Println(err)
	}

	oo := strings.Split(o, "no")
	if strings.TrimSpace(oo[1]) == "" {
		fmt.Println("bridge is reporting nothing else in it")
		return true
	}

	return false
}
