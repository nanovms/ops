package cmd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// TODO: Remove hard coding
const bridgeName string = "br0"
const tapName string = "tap0"
const macAddr string = "08-00-27-00-A8-E8"

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
	if err = netlink.LinkSetMaster(eth0, bridge); err != nil {
		return nil, err
	}
	if err = addTapDevice(tapName, bridge); err != nil {
		return nil, err
	}
	return bridge, nil
}

func findFirstActiveAdapter() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var eth0 *net.Interface
	for _, eif := range interfaces {
		if eif.Flags&net.FlagUp > 0 {
			// ugly: Need a better way to pick wired ethernet adapter.
			if eif.Name[0] == 'e' {
				eth0 = &eif
				break
			}
		}
	}
	if eth0 == nil {
		return nil, errors.New("No active ethernet adapter found")
	}
	return eth0, nil
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

func randomMacAddress() net.HardwareAddr {
	mac := make([]byte, 6)
	rand.Read(mac)
	// local bit
	mac[0] |= 2
	return net.HardwareAddr{mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]}
}
