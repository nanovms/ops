package lepton

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	compute "google.golang.org/api/compute/v1"
)

// CreateVPC creates a legacy virtual network with the name specified
func (p *GCloud) CreateVPC(computeService *compute.Service, project string, name string) (network *compute.Network, err error) {
	networkPayload := &compute.Network{
		Name:                  name,
		AutoCreateSubnetworks: false,
	}

	createOperation, err := computeService.Networks.Insert(project, networkPayload).Do()
	if err != nil {
		return
	}

	err = p.pollOperation(context.TODO(), project, computeService, *createOperation)
	if err != nil {
		return
	}

	network, err = p.GetVPC(computeService, project, name)

	return
}

// GetVPC returns the vpc with the name specified
func (p *GCloud) GetVPC(computeService *compute.Service, project string, name string) (network *compute.Network, err error) {
	network, err = computeService.Networks.Get(project, name).Do()

	return
}

func (p *GCloud) findOrCreateVPC(ctx *Context, computeService *compute.Service, vpcName string) (network *compute.Network, err error) {
	c := ctx.config

	network, err = p.GetVPC(computeService, c.CloudConfig.ProjectID, vpcName)
	if err != nil && strings.Contains(err.Error(), "notFound") {
		ctx.logger.Warn(err.Error())

		ctx.logger.Info("Creating vpc with name %s", vpcName)
		network, err = p.CreateVPC(computeService, c.CloudConfig.ProjectID, vpcName)
		if err != nil {
			ctx.logger.Error(err.Error())
			err = fmt.Errorf("failed creating vpc %s", vpcName)
			return
		}
		ctx.logger.Info("vpc created")

	} else if err != nil {
		ctx.logger.Error(err.Error())
		err = fmt.Errorf("failed getting vpc %s", vpcName)
	}

	return
}

// CreateSubnet creates a subnet with the name specified
// TODO: Specify required subnet IpCidrRange without overlapping other subnetworks ip range.
// Requires fetching every subnet and find an unused Ip range
func (p *GCloud) CreateSubnet(computeService *compute.Service, project string, region string, name string, vpc *compute.Network) (network *compute.Subnetwork, err error) {
	subnetPayload := &compute.Subnetwork{
		Name:    name,
		Region:  region,
		Network: vpc.Name,
		//IpCidrRange: ,
	}

	createOperation, err := computeService.Subnetworks.Insert(project, region, subnetPayload).Do()
	if err != nil {
		return
	}

	err = p.pollOperation(context.TODO(), project, computeService, *createOperation)
	if err != nil {
		return
	}

	network, err = p.GetSubnet(computeService, project, region, vpc.SelfLink, name)

	return
}

// GetSubnet returns the subnet in vpc with the name specified
func (p *GCloud) GetSubnet(computeService *compute.Service, project string, region string, vpc string, name string) (subnet *compute.Subnetwork, err error) {
	subnet, err = computeService.Subnetworks.Get(project, region, name).Do()

	if err == nil && subnet.Network != vpc {
		subnet = nil
		err = fmt.Errorf("subnet %s in vpc %s not found: notFound", vpc, name)
	}

	return
}

// getNIC uses context configuration to return the network interface controller with the network and subnetwork specified or the default network interface controller
func (p *GCloud) getNIC(ctx *Context, computeService *compute.Service) (nic []*compute.NetworkInterface, err error) {
	c := ctx.config

	var network *compute.Network

	vpcName := c.RunConfig.VPC

	if vpcName != "" {
		network, err = p.findOrCreateVPC(ctx, computeService, vpcName)
		if err != nil {
			return
		}

		var subnet *compute.Subnetwork
		subnetName := c.RunConfig.Subnet

		if subnetName != "" {
			regionParts := strings.Split(c.CloudConfig.Zone, "-")
			region := strings.Join(regionParts[0:2], "-")

			subnet, err = p.GetSubnet(computeService, c.CloudConfig.ProjectID, region, network.SelfLink, subnetName)
			if err != nil && strings.Contains(err.Error(), "notFound") {
				err = fmt.Errorf("make sure you have subnet \"%s\" under vpc \"%s\" in region \"%s\"", subnetName, vpcName, region)
				return
			} else if err != nil {
				ctx.logger.Error(err.Error())
				err = fmt.Errorf("failed getting subnet %s", subnetName)
				return
			}

			nic = append(nic, &compute.NetworkInterface{
				Name:       "eth0",
				Network:    network.SelfLink,
				Subnetwork: subnet.SelfLink,
			})
		} else {
			nic = append(nic, &compute.NetworkInterface{
				Name:    "eth0",
				Network: network.SelfLink,
			})
		}
	} else {
		nic = append(nic, &compute.NetworkInterface{
			Name: "eth0",
			AccessConfigs: []*compute.AccessConfig{
				{
					NetworkTier: "PREMIUM",
					Type:        "ONE_TO_ONE_NAT",
					Name:        "External NAT",
				},
			},
		})
	}

	return
}

func (p *GCloud) buildFirewallRule(protocol string, ports []int, tag string) *compute.Firewall {
	var portsStr []string
	for _, i := range ports {
		portsStr = append(portsStr, strconv.Itoa(i))
	}

	return &compute.Firewall{
		Name:        fmt.Sprintf("ops-%s-rule-%s", protocol, tag),
		Description: fmt.Sprintf("Allow traffic to %v ports %s", arrayToString(ports, "[]"), tag),
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: protocol,
				Ports:      portsStr,
			},
		},
		TargetTags:   []string{tag},
		SourceRanges: []string{"0.0.0.0/0"},
	}
}
