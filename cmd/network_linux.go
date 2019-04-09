package cmd

import (
	"net"

	"github.com/d2g/dhcp4"
	"github.com/d2g/dhcp4client"
	"github.com/go-errors/errors"
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

func dhcpAcquireIPAddress(mac string) (string, error) {
	var err error
	// we might need to cache mac before we can
	// use random mac.
	//m :=  randomMacAddress()
	m, _ := net.ParseMAC(mac)
	//Create a connection to use
	c, err := dhcp4client.NewPacketSock(2)
	if err != nil {
		return "", err
	}
	defer c.Close()

	dhcpClient, err := dhcp4client.New(dhcp4client.HardwareAddr(m), dhcp4client.Connection(c))
	if err != nil {
		return "", err
	}
	defer dhcpClient.Close()

	discoveryPacket, err := dhcpClient.SendDiscoverPacket()
	if err != nil {
		return "", err
	}

	offerPacket, err := dhcpClient.GetOffer(&discoveryPacket)
	if err != nil {
		return "", err
	}

	requestPacket, err := dhcpClient.SendRequest(&offerPacket)
	if err != nil {
		return "", err
	}

	acknowledgementpacket, err := dhcpClient.GetAcknowledgement(&requestPacket)
	if err != nil {
		return "", err
	}

	acknowledgementOptions := acknowledgementpacket.ParseOptions()
	if dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK {
		return "", errors.New("ACK not received")
	}
	return acknowledgementpacket.YIAddr().String(), nil
}
