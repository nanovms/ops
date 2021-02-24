package cmd

import (
	"fmt"

	"github.com/nanovms/ops/network"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"
)

// RunLocalInstance runs a virtual machine in a hypervisor
func RunLocalInstance(c *types.Config) (err error) {
	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		ErrNoHypervisor := "No hypervisor found on $PATH"
		InfoInstallOps := "Please install OPS using curl https://ops.city/get.sh -sSfL | sh"
		return fmt.Errorf("%s\n%s", ErrNoHypervisor, InfoInstallOps)
	}

	tapDeviceName := c.RunConfig.TapName
	bridged := c.RunConfig.Bridged
	ipaddr := c.RunConfig.IPAddr
	netmask := c.RunConfig.NetMask

	bridgeName := c.RunConfig.BridgeName
	if bridged && bridgeName == "" {
		bridgeName = "br0"
	}

	networkService := network.NewIprouteNetworkService()

	if tapDeviceName != "" {
		err = network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, ipaddr, netmask)
		if err != nil {
			return
		}
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)
	hypervisor.Start(&c.RunConfig)

	if tapDeviceName != "" {
		err = network.TurnOffNetworkInterfaces(networkService, tapDeviceName, bridgeName)
		if err != nil {
			return
		}
	}

	return
}
