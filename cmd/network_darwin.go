package cmd

import (
	"fmt"
	"github.com/vishvananda/netlink"
)

func addTapDevice(name string, bridge *netlink.Bridge) error {
	return nil
}

func dhcpAcquireIPAddress(mac string) (string, error) {
	return "", nil
}

func setupBridgeNetwork() error {
	return nil
}

func createBridgeNetwork(adapterName string) (*netlink.Bridge, error) {
	// create a bridge named br0
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	bridge := &netlink.Bridge{LinkAttrs: la}
	err := netlink.LinkAdd(bridge)
	if err != nil {
		return nil, err
	}
	// bring up the bridge
	if err = netlink.LinkSetUp(bridge); err != nil {
		return nil, err
	}
	// add network adapter to bridge
	eth0, err := netlink.LinkByName(adapterName)
	if err != nil {
		return nil, err
	}

	if err = netlink.LinkSetMaster(eth0, bridge); err != nil {
		return nil, err
	}
	if err = addTapDevice(tapName, bridge); err != nil {
		return nil, err
	}
	return bridge, nil
}

func assignIP(bridge *netlink.Bridge) error {
	var err error
	ip, err := dhcpAcquireIPAddress(macAddr)
	if err != nil {
		return err
	}
	addr, err := netlink.ParseAddr(fmt.Sprintf("%v/%v", ip, 24))
	if err != nil {
		return err
	}
	netlink.AddrAdd(bridge, addr)
	return nil
}

func resetBridgeNetwork() error {
	eth0, err := findFirstActiveAdapter()
	if err != nil {
		return err
	}

	adapterName := eth0.Name
	br0, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	adapter, err := netlink.LinkByName(adapterName)
	if err != nil {
		return err
	}

	if err = netlink.LinkSetNoMaster(adapter); err != nil {
		return err
	}

	tap0, err := netlink.LinkByName(tapName)
	if err != nil {
		return err
	}

	if err = netlink.LinkSetNoMaster(tap0); err != nil {
		return err
	}

	if err = netlink.LinkSetDown(tap0); err != nil {
		return err
	}

	if err = netlink.LinkSetDown(br0); err != nil {
		return err
	}

	if err = netlink.LinkDel(tap0); err != nil {
		return err
	}

	if err = netlink.LinkDel(br0); err != nil {
		return err
	}

	return nil
}
