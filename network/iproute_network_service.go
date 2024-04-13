package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// IprouteNetworkService uses iproute commands to change network configuration in OS
type IprouteNetworkService struct {
}

func execCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	output = string(out)
	return
}

// NewIprouteNetworkService returns an instance of IprouteNetworkService
func NewIprouteNetworkService() *IprouteNetworkService {
	return &IprouteNetworkService{}
}

// AddBridge creates bridge interface
func (s *IprouteNetworkService) AddBridge(br string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip link add name %s type bridge", br))
}

// ListBridges prints a list of bridge network interfaces
func (s *IprouteNetworkService) ListBridges() (string, error) {
	return execCmd("sudo ip link show type bridge")
}

// CheckBridgeHasInterface checks whether interface is listed in bridge network
func (s *IprouteNetworkService) CheckBridgeHasInterface(bridgeName string, ifcName string) (bool, error) {
	output, err := execCmd(fmt.Sprintf("sudo ip link show master %s", bridgeName))
	if err != nil {
		return false, err
	}

	ifcs := strings.Split(output, string('\t'))

	for i := 0; i < len(ifcs); i++ {
		words := strings.Split(ifcs[i], string(' '))

		if len(words) > 2 {
			if strings.Replace(words[1], ":", "", 1) == ifcName {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetBridgeInterfacesNames get list of interfaces names in bridge network
func (s *IprouteNetworkService) GetBridgeInterfacesNames(bridgeName string) ([]string, error) {
	output, err := execCmd(fmt.Sprintf("sudo ip link show master %s", bridgeName))
	if err != nil {
		return nil, err
	}

	ifcs := strings.Split(output, string('\n'))

	ifcsNames := []string{}

	for i := 0; i < len(ifcs); i += 2 {
		words := strings.Split(ifcs[i], string(' '))

		if len(words) > 2 {
			ifcsNames = append(ifcsNames, strings.Replace(words[1], ":", "", 1))
		}
	}

	return ifcsNames, nil
}

// CheckNetworkInterfaceExists checks whether network interface exists
func (s *IprouteNetworkService) CheckNetworkInterfaceExists(name string) (bool, error) {
	ifcs, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, i := range ifcs {
		if i.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// AddTap creates tap interface
func (s *IprouteNetworkService) AddTap(tap string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip tuntap add %s mode tap", tap))
}

// AddTapToBridge adds tap interface to bridge network
func (s *IprouteNetworkService) AddTapToBridge(br, tap string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip link set %s master %s", tap, br))
}

// SetNIIP sets network interface IP
func (s *IprouteNetworkService) SetNIIP(ifc string, ip string, netmask string) (string, error) {
	cmd := fmt.Sprintf("sudo ip address add %s/%s dev %s", ip, netmask, ifc)

	return execCmd(cmd)
}

// FlushIPFromNI removes every IP assigned to network interface
func (s *IprouteNetworkService) FlushIPFromNI(niName string) (string, error) {
	cmd := fmt.Sprintf("sudo ip address flush dev %s", niName)

	return execCmd(cmd)
}

// TurnNIUp turns on network interface
func (s *IprouteNetworkService) TurnNIUp(ifc string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip link set dev %s up", ifc))
}

// TurnNIDown turns off network interface
func (s *IprouteNetworkService) TurnNIDown(ifc string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip link set dev %s down", ifc))
}

// DeleteNIC removes network interface
func (s *IprouteNetworkService) DeleteNIC(ifc string) (string, error) {
	return execCmd(fmt.Sprintf("sudo ip link delete %s", ifc))
}

// IsNIUp checks whether network interface is on
func (s *IprouteNetworkService) IsNIUp(ifcName string) (bool, error) {
	output, err := execCmd("sudo ip link ls up")
	if err != nil {
		return false, err
	}

	ifcs := strings.Split(output, string('\n'))

	for i := 0; i < len(ifcs); i += 2 {
		parts := strings.Split(ifcs[i], string(' '))

		if len(parts) > 2 {
			if strings.Replace(parts[1], ":", "", 1) == ifcName {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetNetworkInterfaceIP get IP from network interface
func (s *IprouteNetworkService) GetNetworkInterfaceIP(ifcName string) (string, error) {
	output, err := execCmd(fmt.Sprintf("sudo ip address show %s", ifcName))
	if err != nil {
		return "", err
	}

	parts := strings.Split(output, string(' '))

	if len(parts) != 0 {
		for i, p := range parts {

			if p == "inet" && i < len(parts)-1 {
				slashIndex := strings.Index(parts[i+1], "/")
				return parts[i+1][:slashIndex], nil
			}

		}
	}

	return "", nil
}
