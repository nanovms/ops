//go:build oci || !onlyprovider

package oci

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

// GetSubnet returns a public subnet
func (p *Provider) GetSubnet(ctx *lepton.Context) (subnet *core.Subnet, err error) {
	listSubnetsResponse, err := p.networkClient.ListSubnets(context.TODO(), core.ListSubnetsRequest{CompartmentId: &p.compartmentID})
	if err != nil {
		return
	}

	for _, s := range listSubnetsResponse.Items {
		if *s.ProhibitPublicIpOnVnic != true {
			subnet = &s
		}
	}

	if subnet == nil {
		err = errors.New("no public subnets found")
	}

	return
}

// CreateNetworkSecurityGroup creates a network security group with firewall rules
func (p *Provider) CreateNetworkSecurityGroup(ctx *lepton.Context, vcnID string) (sg *core.NetworkSecurityGroup, err error) {
	instanceName := ctx.Config().RunConfig.InstanceName
	var sgResponse core.CreateNetworkSecurityGroupResponse

	sgResponse, err = p.networkClient.CreateNetworkSecurityGroup(context.TODO(), core.CreateNetworkSecurityGroupRequest{
		CreateNetworkSecurityGroupDetails: core.CreateNetworkSecurityGroupDetails{
			CompartmentId: &p.compartmentID,
			VcnId:         &vcnID,
			DisplayName:   types.StringPtr(instanceName + "-sg"),
		},
	})
	if err != nil {
		ctx.Logger().Error(err)
		err = errors.New("failed creating security group")
		return
	}

	sg = &sgResponse.NetworkSecurityGroup

	sgRules := []core.AddSecurityRuleDetails{}

	if len(ctx.Config().RunConfig.Ports) > 0 {
		var tcpRules []core.AddSecurityRuleDetails

		tcpRules, err = p.addNetworkSecurityRules(ctx.Config().RunConfig.Ports, "tcp")
		if err != nil {
			ctx.Logger().Log(err.Error())
			err = errors.New("failed adding tcp rules")
			return
		}

		sgRules = append(sgRules, tcpRules...)
	}

	if len(ctx.Config().RunConfig.UDPPorts) > 0 {
		var udpRules []core.AddSecurityRuleDetails

		udpRules, err = p.addNetworkSecurityRules(ctx.Config().RunConfig.UDPPorts, "udp")
		if err != nil {
			ctx.Logger().Log(err.Error())
			err = errors.New("failed adding udp rules")
			return
		}

		sgRules = append(sgRules, udpRules...)
	}

	if len(sgRules) > 0 {
		_, err = p.networkClient.AddNetworkSecurityGroupSecurityRules(context.TODO(), core.AddNetworkSecurityGroupSecurityRulesRequest{
			NetworkSecurityGroupId: sg.Id,
			AddNetworkSecurityGroupSecurityRulesDetails: core.AddNetworkSecurityGroupSecurityRulesDetails{
				SecurityRules: sgRules,
			},
		})
		if err != nil {
			err = fmt.Errorf("failed adding rules to security group: %s", err)
		}
	}

	return
}

func (p *Provider) addNetworkSecurityRules(ports []string, protocol string) (sgRules []core.AddSecurityRuleDetails, err error) {
	for _, port := range ports {
		var min, max int
		min, max, err = parsePorts(port)
		if err != nil {
			return
		}

		ingressRule := core.AddSecurityRuleDetails{
			Direction:       core.AddSecurityRuleDetailsDirectionIngress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Source:          types.StringPtr("0.0.0.0/0"),
		}
		egressRule := core.AddSecurityRuleDetails{
			Direction:       core.AddSecurityRuleDetailsDirectionEgress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Destination:     types.StringPtr("0.0.0.0/0"),
		}

		if protocol == "tcp" {
			ingressRule.Protocol = types.StringPtr("6")
			egressRule.Protocol = types.StringPtr("6")
			ingressRule.TcpOptions = &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: &min,
					Max: &max,
				},
			}
			egressRule.TcpOptions = &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(1),
					Max: common.Int(65535),
				},
			}
		} else if protocol == "udp" {
			ingressRule.Protocol = types.StringPtr("17")
			egressRule.Protocol = types.StringPtr("17")
			ingressRule.UdpOptions = &core.UdpOptions{
				DestinationPortRange: &core.PortRange{
					Min: &min,
					Max: &max,
				},
			}
			egressRule.UdpOptions = &core.UdpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(1),
					Max: common.Int(65535),
				},
			}
		} else {
			err = errors.New("invalid protocol")
			return
		}

		sgRules = append(sgRules, ingressRule)
		sgRules = append(sgRules, egressRule)
	}

	return

}

func parsePorts(port string) (min int, max int, err error) {
	portsParts := strings.Split(port, "-")
	if len(portsParts) == 1 {
		min, err = strconv.Atoi(portsParts[0])
		if err != nil {
			err = fmt.Errorf("failed parsing port %s: %s", portsParts[0], err)
			return
		}
		max = min
	} else if len(portsParts) == 2 {
		min, err = strconv.Atoi(portsParts[0])
		if err != nil {
			err = fmt.Errorf("failed parsing port %s, %s", portsParts[0], err)
			return
		}
		max, err = strconv.Atoi(portsParts[1])
		if err != nil {
			err = fmt.Errorf("failed parsing port %s: %s", portsParts[0], err)
			return
		}
	}
	return
}
