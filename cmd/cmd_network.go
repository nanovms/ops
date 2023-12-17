// currently for linux only and relies on bridgetools and dnsmasq

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NetworkCommands have the capability of creating/destroying networks.
// linux only for now - mac uses vmnet-bridged.
func NetworkCommands() *cobra.Command {
	var cmdNetwork = &cobra.Command{
		Use:       "network",
		Short:     "manage nanos network",
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

	cmdNetworkCreate.PersistentFlags().StringP("bridgename", "", "", "bridge name")
	cmdNetworkCreate.PersistentFlags().StringP("subnet", "", "", "subnet")

	return cmdNetworkCreate
}

func networkCreateCommandHandler(cmd *cobra.Command, args []string) {
	bn, err := cmd.Flags().GetString("bridgename")
	if err != nil {
		fmt.Println(err)
	}

	subnet, err := cmd.Flags().GetString("subnet")
	if err != nil {
		fmt.Println(err)
	}

	bridge := "br0"
	if bn != "" {
		bridge = bn
	}

	// mv me elsewhere

	// option; also break out class-c to provide range
	network := "192.168.1.1/24"
	if subnet != "" {
		network = subnet
	}

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

		// stubbed to /24 for now
		rz := strings.Split(network, "/")
		rz = strings.Split(rz[0], ".")
		tzr := rz[0] + "." + rz[1] + "." + rz[2]

		ecmd = exec.Command("sudo", "dnsmasq", "--bind-interfaces", "--interface="+bridge, "--except-interface=lo", "--leasefile-ro", "--dhcp-range="+tzr+".2,"+tzr+".251,12")
		out, err = ecmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(string(out))
	}

	opshome := lepton.GetOpsHome()
	networks := path.Join(opshome, "networks")

	n := Network{
		Network: network,
		Name:    bridge,
	}

	d1, err := json.Marshal(n)
	if err != nil {
		fmt.Println(err)
	}

	err = os.WriteFile(networks+"/"+n.Name, d1, 0644)
	if err != nil {
		fmt.Println(err)
	}

}

// Network contains individual user-created network details.
type Network struct {
	Network string
	Name    string
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
	// context me
	opshome := lepton.GetOpsHome()
	networksPath := path.Join(opshome, "networks")
	files, err := os.ReadDir(networksPath)
	if err != nil {
		fmt.Println(err)
	}

	networks := []Network{}

	for _, f := range files {
		fpath := path.Join(networksPath, f.Name())

		body, err := os.ReadFile(fpath)
		if err != nil {
			fmt.Println(err)
		}

		var n Network
		err = json.Unmarshal(body, &n)
		if err != nil {
			fmt.Println(err)
		}

		networks = append(networks, n)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Network"})
	table.SetRowLine(true)

	for _, i := range networks {
		var rows []string
		rows = append(rows, i.Name)
		rows = append(rows, i.Network)

		table.Append(rows)
	}

	table.Render()
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
	killBridge(args[0])

	opshome := lepton.GetOpsHome()
	ipath := path.Join(opshome, "networks", args[0])
	err := os.Remove(ipath)
	if err != nil {
		fmt.Println(err)
	}

}

// mv elsewhere and get rid of shelling

func killBridge(bridgeName string) {
	log := false

	if log {
		fmt.Printf("killing bridge %s\n", bridgeName)
	}

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
