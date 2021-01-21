package cmd

import (
	"fmt"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/network"
)

// StartUnikernel runs a virtual machine in a hypervisor
func StartUnikernel(c *api.Config) (err error) {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		ErrNoHypervisor := "No hypervisor found on $PATH"
		InfoInstallOps := "Please install OPS using curl https://ops.city/get.sh -sSfL | sh"
		return fmt.Errorf("%s\n%s", ErrNoHypervisor, InfoInstallOps)
	}

	tapDeviceName := c.RunConfig.TapName
	bridged := c.RunConfig.Bridged
	ipaddr := c.RunConfig.IPAddr
	netmask := c.RunConfig.NetMask
	bridgeName := "br0"

	networkService := network.NewIprouteNetworkService()

	if tapDeviceName != "" {
		err = network.SetupNetworkInterfaces(networkService, tapDeviceName, bridged, ipaddr, netmask)
		if err != nil {
			return
		}
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.Imagename)
	hypervisor.Start(&c.RunConfig)

	if tapDeviceName != "" {
		err = network.TurnOffNetworkInterfaces(networkService, tapDeviceName, bridged, bridgeName)
		if err != nil {
			return
		}
	}

	return
}
