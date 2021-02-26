package network

import (
	"bytes"
	"net"
)

// AllocateNewCidrBlock returns an unused cidr block
func AllocateNewCidrBlock(existingCidrBlocks []string) string {
	var maxIPNet *net.IPNet
	var maxIP net.IP

	for _, cidrBlock := range existingCidrBlocks {
		if maxIPNet == nil {
			maxIP, maxIPNet, _ = net.ParseCIDR(cidrBlock)
		}

		ip, ipNet, _ := net.ParseCIDR(cidrBlock)

		if bytes.Compare(ip, maxIP) > 0 {
			maxIPNet = ipNet
			maxIP = ip
		}
	}

	if maxIPNet == nil {
		return ""
	}

	netmaskBits, _ := maxIPNet.Mask.Size()

	if netmaskBits == 16 {
		maxIPNet.IP[1]++
		if maxIPNet.IP[1] == 0 {
			maxIPNet.IP[0]++
		}
		return maxIPNet.String()
	} else if netmaskBits == 24 {
		maxIPNet.IP[2]++
		if maxIPNet.IP[2] == 0 {
			maxIPNet.IP[1]++
		}
		if maxIPNet.IP[1] == 0 {
			maxIPNet.IP[0]++
		}
		return maxIPNet.String()
	}

	return ""
}
