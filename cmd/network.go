package cmd

import (
	"crypto/rand"
	"errors"
	"net"
)

// TODO: Remove hard coding
const bridgeName string = "br0"
const tapName string = "tap0"
const macAddr string = "08-00-27-00-A8-E8"

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

func randomMacAddress() net.HardwareAddr {
	mac := make([]byte, 6)
	rand.Read(mac)
	// local bit
	mac[0] |= 2
	return net.HardwareAddr{mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]}
}
