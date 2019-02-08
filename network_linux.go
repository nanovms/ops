package main

import (
	"github.com/vishvananda/netlink"
)

func addTapDevice(name string, bridge *netlink.Bridge) error {
	tap0 := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{Name: name},
		Mode:      netlink.TUNTAP_MODE_TAP,
	}
	if err := netlink.LinkAdd(tap0); err != nil {
		return err
	}
	// bring up tap0
	if err := netlink.LinkSetUp(tap0); err != nil {
		return err
	}
	if err := netlink.LinkSetMaster(tap0, bridge); err != nil {
		return err
	}
	return nil
}

func setupBridgeNetwork() error {
	var err error
	eth0, err := findFirstActiveAdapter()
	if err != nil {
		return err
	}
	bridge, err := createBridgeNetwork(eth0.Name)
	if err != nil {
		return err
	}
	err = assignIP(bridge)
	if err != nil {
		return err
	}
	return nil
}
