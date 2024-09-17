package cmd

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/network"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"
)

// RunLocalInstance runs a virtual machine in a hypervisor
func RunLocalInstance(c *types.Config) (err error) {
	if c.Mounts != nil {
		c.VolumesDir = lepton.LocalVolumeDir

		err = onprem.AddMountsFromConfig(c)
		if err != nil {
			return
		}
	}
	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		ErrNoHypervisor := "No hypervisor found on $PATH"
		InfoInstallOps := "Please install OPS using curl https://ops.city/get.sh -sSfL | sh"
		return fmt.Errorf("%s\n%s", ErrNoHypervisor, InfoInstallOps)
	}

	tapDeviceName := c.RunConfig.TapName
	bridged := c.RunConfig.Bridged
	ipaddress := c.RunConfig.IPAddress
	bridgeipaddress := c.RunConfig.BridgeIPAddress
	netmask := c.RunConfig.NetMask

	bridgeName := c.RunConfig.BridgeName
	if bridged && bridgeName == "" {
		bridgeName = "br0"
	}

	networkService := network.NewIprouteNetworkService()

	if tapDeviceName != "" {
		err = network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, ipaddress, netmask, bridgeipaddress)
		if err != nil {
			return
		}
	}

	fmt.Println("running local instance")

	c.RunConfig.Kernel = c.Kernel

	if c.RunConfig.QMP {
		c.RunConfig.Mgmt = qemu.GenMgmtPort()
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.ImageName)
	hypervisor.Start(&c.RunConfig)

	if tapDeviceName != "" {
		err = network.TurnOffNetworkInterfaces(networkService, tapDeviceName, bridgeName)
		if err != nil {
			return
		}
	}

	return
}
