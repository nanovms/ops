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

// NewIprouteNetworkService returns an instance of IprouteNetworkService
func NewIprouteNetworkService() *IprouteNetworkService {
	return &IprouteNetworkService{}
}

// AddBridge creates bridge interface
func (s *IprouteNetworkService) AddBridge(br string) (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "add", "name", br, "type", "bridge")
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err

}

// ListBridges prints a list of bridge network interfaces
func (s *IprouteNetworkService) ListBridges() (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "show", "type", "bridge")
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// CheckBridgeHasInterface checks whether interface is listed in bridge network
func (s *IprouteNetworkService) CheckBridgeHasInterface(bridgeName string, ifcName string) (bool, error) {
	cmd := exec.Command("sudo", "ip", "link", "show", "master", bridgeName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	output := string(out)

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
	cmd := exec.Command("sudo", "ip", "link", "show", "master", bridgeName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	output := string(out)

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
	cmd := exec.Command("sudo", "ip", "tuntap", "add", tap, "mode", "tap")
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// AddTapToBridge adds tap interface to bridge network
func (s *IprouteNetworkService) AddTapToBridge(br, tap string) (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "set", tap, "master", br)
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// SetNIIP sets network interface IP
func (s *IprouteNetworkService) SetNIIP(ifc string, ip string, netmask string) (string, error) {
	ipn := fmt.Sprintf("%s/%s", ip, netmask)
	cmd := exec.Command("sudo", "ip", "address", "add", ipn, "dev", ifc)
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// FlushIPFromNI removes every IP assigned to network interface
func (s *IprouteNetworkService) FlushIPFromNI(niName string) (string, error) {
	cmd := exec.Command("sudo", "ip", "address", "flush", "dev", niName)
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// TurnNIUp turns on network interface
func (s *IprouteNetworkService) TurnNIUp(ifc string) (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "set", "dev", ifc, "up")
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// TurnNIDown turns off network interface
func (s *IprouteNetworkService) TurnNIDown(ifc string) (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "set", "dev", ifc, "down")
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// DeleteNIC removes network interface
func (s *IprouteNetworkService) DeleteNIC(ifc string) (string, error) {
	cmd := exec.Command("sudo", "ip", "link", "delete", ifc)
	out, err := cmd.CombinedOutput()
	output := string(out)
	return output, err
}

// IsNIUp checks whether network interface is on
func (s *IprouteNetworkService) IsNIUp(ifcName string) (bool, error) {
	cmd := exec.Command("sudo", "ip", "link", "ls", "up")
	out, err := cmd.CombinedOutput()
	output := string(out)
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
	cmd := exec.Command("sudo", "ip", "address", "show", ifcName)
	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}
	output := string(out)

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
