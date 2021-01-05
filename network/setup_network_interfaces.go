package network

import (
	"errors"
	"net"
)

// Service represents a network service able to apply changes to network configuration
type Service interface {
	AddBridge(br string) (string, error)
	ListBridges() (string, error)
	CheckBridgeHasInterface(bridgeName string, ifcName string) (bool, error)
	GetBridgeInterfacesNames(bridgeName string) ([]string, error)
	CheckNetworkInterfaceExists(name string) (bool, error)
	AddTap(tap string) (string, error)
	AddTapToBridge(br, tap string) (string, error)
	SetNIIP(ifc string, ip string, netmask string) (string, error)
	FlushIPFromNI(niName string) (string, error)
	TurnNIUp(ifc string) (string, error)
	TurnNIDown(ifc string) (string, error)
	DeleteNIC(ifc string) (string, error)
	IsNIUp(ifcName string) (bool, error)
	GetNetworkInterfaceIP(ifcName string) (string, error)
}

// SetupNetworkInterfaces changes network configuration to support requirements
func SetupNetworkInterfaces(network Service, tapDeviceName string, bridged bool, ipaddr string, netmask string) error {
	tapExists, err := network.CheckNetworkInterfaceExists(tapDeviceName)
	if err != nil {
		return errors.New("Not able to check tap exists")
	}

	if !tapExists {
		_, err := network.AddTap(tapDeviceName)
		if err != nil {
			return errors.New("Not able to create tap")
		}
	}

	bridgeName := "br0"

	if bridged {

		bridgeExists, err := network.CheckNetworkInterfaceExists(bridgeName)
		if err != nil {
			return errors.New("Not able to check if bridge exists")
		}

		if !bridgeExists {
			_, err := network.AddBridge(bridgeName)
			if err != nil {
				return errors.New("Not able to create bridge")
			}
		}

		if ipaddr != "" {
			ip := net.ParseIP(ipaddr).To4()
			ip[3] = byte(66) // 66 is a random IP
			bridgeIP := ip.String()

			currentBridgeIP, err := network.GetNetworkInterfaceIP(bridgeName)
			if err != nil {
				return errors.New("Not able to get current bridge IP")
			}

			if currentBridgeIP != bridgeIP {
				_, err = network.FlushIPFromNI(bridgeName)
				if err != nil {
					return errors.New("Not able to flush bridge IPs")
				}

				_, err = network.SetNIIP(bridgeName, bridgeIP, netmask)
				if err != nil {
					return errors.New("Not able to assign IP to bridge")
				}
			}

		}

		isTapInTheBridge, err := network.CheckBridgeHasInterface(bridgeName, tapDeviceName)
		if err != nil {
			return errors.New("Not able to check if tap is in the bridge")
		}

		if !isTapInTheBridge {
			_, err := network.AddTapToBridge(bridgeName, tapDeviceName)
			if err != nil {
				return errors.New("Not able to add tap to bridge")
			}
		}

		isBridgeUp, err := network.IsNIUp(bridgeName)
		if err != nil {
			return errors.New("Not able to check if bridge is up")
		}

		if !isBridgeUp {
			_, err := network.TurnNIUp(bridgeName)
			if err != nil {
				return errors.New("Not able to turn bridge up")
			}
		}
	}

	isTapUp, err := network.IsNIUp(tapDeviceName)
	if err != nil {
		return errors.New("Not able to check if tap is up")
	}

	if !isTapUp {
		_, err = network.TurnNIUp(tapDeviceName)
		if err != nil {
			return errors.New("Not able to turn tap up")
		}
	}

	return nil
}

// TurnOffNetworkInterfaces turns network interfaces off if they aren't used
func TurnOffNetworkInterfaces(network Service, tapDeviceName string, bridged bool, bridgeName string) error {
	_, err := network.TurnNIDown(tapDeviceName)
	if err != nil {
		return errors.New("Not able to turn tap down")
	}

	if bridged {
		bridgeInterfaces, err := network.GetBridgeInterfacesNames(bridgeName)
		if err != nil {
			return errors.New("Not able to get network interfaces in bridge")
		}

		var someBridgeInterfaceIsUp bool

		for _, bi := range bridgeInterfaces {
			isUp, err := network.IsNIUp(bi)
			if err != nil {
				return errors.New("Not able to get interface state")
			}

			if isUp {
				someBridgeInterfaceIsUp = true
				break
			}
		}

		if !someBridgeInterfaceIsUp {
			_, err := network.TurnNIDown(bridgeName)
			if err != nil {
				return errors.New("Not able to turn bridge down")
			}

			_, err = network.FlushIPFromNI(bridgeName)
			if err != nil {
				return errors.New("Not able to flush IPs from bridge")
			}
		}

	}

	return nil
}
