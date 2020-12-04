package lepton

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func getAzureResourceNameFromID(id string) string {
	idParts := strings.Split(id, "/")
	return idParts[len(idParts)-1]
}

func (a *Azure) getNicClient() *network.InterfacesClient {
	nicClient := network.NewInterfacesClient(a.subID)
	nicClient.Authorizer = *a.authorizer
	nicClient.AddToUserAgent(userAgent)
	return &nicClient
}

// CreateNIC creates a new network interface. The Network Security Group
// is not a required parameter
func (a *Azure) CreateNIC(ctx context.Context, location string, vnetName, subnetName, nsgName, ipName, nicName string) (nic network.Interface, err error) {
	subnet, err := a.GetVirtualNetworkSubnet(ctx, vnetName, subnetName)
	if err != nil {
		log.Fatalf("failed to get subnet: %v", err)
	}

	ip, err := a.GetPublicIP(ctx, ipName)
	if err != nil {
		log.Fatalf("failed to get ip address: %v", err)
	}

	nicParams := network.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(location),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
		Tags: getAzureDefaultTags(),
	}

	if nsgName != "" {
		nsg, err := a.GetNetworkSecurityGroup(ctx, nsgName)
		if err != nil {
			log.Fatalf("failed to get nsg: %v", err)
		}
		nicParams.NetworkSecurityGroup = nsg
	}

	nicClient := a.getNicClient()

	future, err := nicClient.CreateOrUpdate(ctx, a.groupName, nicName, nicParams)
	if err != nil {
		return nic, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		return nic, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	nic, err = future.Result(*nicClient)
	return
}

// DeleteNIC deletes network interface controller
func (a *Azure) DeleteNIC(ctx *Context, nic *network.Interface) error {
	logger := ctx.logger

	nicClient := a.getNicClient()

	logger.Info("Deleting %s...", *nic.ID)
	nicName := getAzureResourceNameFromID(*nic.ID)
	nicDeleteTask, err := nicClient.Delete(context.TODO(), a.groupName, nicName)
	if err != nil {
		logger.Error(err.Error())
		return errors.New("error deleting network interface controller")
	}

	err = nicDeleteTask.WaitForCompletionRef(context.TODO(), nicClient.Client)
	if err != nil {
		logger.Error(err.Error())
		return errors.New("error waiting for network interface controller deleting")
	}

	return nil
}

// DeletePublicIPs deletes array of ip configurations
func (a *Azure) DeletePublicIPs(ctx *Context, ips *[]network.InterfaceIPConfiguration) error {
	logger := ctx.logger

	ipClient := a.getIPClient()

	for _, ip := range *ips {
		if ip.PublicIPAddress.ID != nil {
			ipID := getAzureResourceNameFromID(*ip.PublicIPAddress.ID)

			logger.Info("Deleting %s...", *ip.PublicIPAddress.ID)
			deleteIPTask, err := ipClient.Delete(context.TODO(), a.groupName, ipID)
			if err != nil {
				logger.Error(err.Error())
				return errors.New("error deleting ip")
			}

			err = deleteIPTask.WaitForCompletionRef(context.TODO(), ipClient.Client)
			if err != nil {
				logger.Error(err.Error())
				return errors.New("error waiting for ip deletion")
			}
		}
	}

	return nil
}

// DeleteNetworkSecurityGroup deletes virtual networks and subnets associated with the security group and the security group itself
func (a *Azure) DeleteNetworkSecurityGroup(ctx *Context, securityGroupID string) error {
	logger := ctx.logger

	nsgClient, err := a.getNsgClient()
	if err != nil {
		return err
	}
	subnetsClient, err := a.getSubnetsClient()
	if err != nil {
		return err
	}
	vnetClient, err := a.getVnetClient()
	if err != nil {
		return err
	}

	securityGroupName := getAzureResourceNameFromID(securityGroupID)
	securityGroup, err := nsgClient.Get(context.TODO(), a.groupName, securityGroupName, "")
	if err != nil {
		logger.Error(err.Error())
		return errors.New("error getting network security group")
	}

	if securityGroup.Subnets != nil {
		for _, subnet := range *securityGroup.Subnets {
			if subnet.ID != nil {
				logger.Info("Deleting %s...", *subnet.ID)
				subnetName := getAzureResourceNameFromID(*subnet.ID)
				subnetDeleteTask, err := subnetsClient.Delete(context.TODO(), a.groupName, subnetName, subnetName)
				if err != nil {
					logger.Error(err.Error())
					return errors.New("error deleting subnet")
				}

				err = subnetDeleteTask.WaitForCompletionRef(context.TODO(), subnetsClient.Client)
				if err != nil {
					logger.Error(err.Error())
					return errors.New("error waiting for subnet deletion")
				}

				logger.Info("Deleting virtualNetworks/%s", subnetName)
				vnDeleteTask, err := vnetClient.Delete(context.TODO(), a.groupName, subnetName)
				if err != nil {
					logger.Error(err.Error())
					return errors.New("error deleting virtual network")
				}

				err = vnDeleteTask.WaitForCompletionRef(context.TODO(), vnetClient.Client)
				if err != nil {
					logger.Error(err.Error())
					return errors.New("error waiting for virtual network deletion")
				}
			}
		}
	}

	logger.Info("Deleting %s...", *securityGroup.ID)
	nsgTask, err := nsgClient.Delete(context.TODO(), a.groupName, *securityGroup.Name)
	if err != nil {
		logger.Error(err.Error())
		return errors.New("error deleting security group")
	}

	err = nsgTask.WaitForCompletionRef(context.TODO(), nsgClient.Client)
	if err != nil {
		logger.Error(err.Error())
		return errors.New("error waiting for security group deletion")
	}

	return nil
}

func (a *Azure) getIPClient() *network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClient(a.subID)
	ipClient.Authorizer = *a.authorizer
	ipClient.AddToUserAgent(userAgent)
	return &ipClient
}

// CreatePublicIP creates a new public IP
func (a *Azure) CreatePublicIP(ctx context.Context, location string, ipName string) (ip network.PublicIPAddress, err error) {
	ipClient := a.getIPClient()

	future, err := ipClient.CreateOrUpdate(
		ctx,
		a.groupName,
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(location),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
			Tags: getAzureDefaultTags(),
		},
	)

	if err != nil {
		return ip, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		return ip, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	ip, err = future.Result(*ipClient)
	return
}

// GetPublicIP returns an existing public IP
func (a *Azure) GetPublicIP(ctx context.Context, ipName string) (ip network.PublicIPAddress, err error) {
	ipClient := a.getIPClient()

	ip, err = ipClient.Get(ctx, a.groupName, ipName, "")
	return
}

func (a *Azure) getVnetClient() (*network.VirtualNetworksClient, error) {
	vnetClient := network.NewVirtualNetworksClient(a.subID)
	authr, err := a.GetResourceManagementAuthorizer()
	if err != nil {
		return nil, err
	}

	vnetClient.Authorizer = authr
	vnetClient.AddToUserAgent(userAgent)
	return &vnetClient, nil
}

// GetVPC finds the virtual network by id
func (a *Azure) GetVPC(vnetName string) (vnet *network.VirtualNetwork, err error) {
	vnetClient, err := a.getVnetClient()
	if err != nil {
		return
	}

	result, err := vnetClient.Get(context.TODO(), a.groupName, vnetName, "")
	vnet = &result
	return
}

// CreateVirtualNetwork creates a virtual network
func (a *Azure) CreateVirtualNetwork(ctx context.Context, location string, vnetName string) (vnet *network.VirtualNetwork, err error) {
	vnetClient, err := a.getVnetClient()
	if err != nil {
		return nil, err
	}
	future, err := vnetClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(location),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
			},
			Tags: getAzureDefaultTags(),
		})

	if err != nil {
		return vnet, fmt.Errorf("cannot create virtual network: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vnetClient.Client)
	if err != nil {
		return vnet, fmt.Errorf("cannot get the vnet create or update future response: %v", err)
	}

	fmt.Printf("\nCreated Virtual Network\n")
	vn, err := future.Result(*vnetClient)

	return &vn, err
}

// CreateVirtualNetworkAndSubnets creates a virtual network with two
// subnets
func (a *Azure) CreateVirtualNetworkAndSubnets(ctx context.Context, location string, vnetName, subnet1Name, subnet2Name string) (vnet *network.VirtualNetwork, err error) {
	vnetClient, err := a.getVnetClient()
	if err != nil {
		return nil, err
	}

	future, err := vnetClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(location),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
				Subnets: &[]network.Subnet{
					{
						Name: to.StringPtr(subnet1Name),
						SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.0.0/16"),
						},
					},
					{
						Name: to.StringPtr(subnet2Name),
						SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.1.0.0/16"),
						},
					},
				},
			},
		})

	if err != nil {
		return vnet, fmt.Errorf("cannot create virtual network: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vnetClient.Client)
	if err != nil {
		return vnet, fmt.Errorf("cannot get the vnet create or update future response: %v", err)
	}

	vn, err := future.Result(*vnetClient)

	return &vn, err
}

func (a *Azure) getSubnetsClient() (*network.SubnetsClient, error) {
	subnetsClient := network.NewSubnetsClient(a.subID)
	auth, err := a.GetResourceManagementAuthorizer()
	if err != nil {
		return nil, err
	}
	subnetsClient.Authorizer = auth
	subnetsClient.AddToUserAgent(userAgent)
	return &subnetsClient, nil
}

// CreateVirtualNetworkSubnet creates a subnet in an existing vnet
func (a *Azure) CreateVirtualNetworkSubnet(ctx context.Context, vnetName, subnetName string) (subnet *network.Subnet, err error) {
	subnetsClient, err := a.getSubnetsClient()
	if err != nil {
		return nil, err
	}

	future, err := subnetsClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		subnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.0.0/16"),
			},
		})
	if err != nil {
		return subnet, fmt.Errorf("cannot create subnet: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, subnetsClient.Client)
	if err != nil {
		return subnet, fmt.Errorf("cannot get the subnet create or update future response: %v", err)
	}

	sc, err := future.Result(*subnetsClient)

	return &sc, err
}

// CreateSubnetWithNetworkSecurityGroup create a subnet referencing a
// network security group
func (a *Azure) CreateSubnetWithNetworkSecurityGroup(ctx context.Context, vnetName, subnetName, addressPrefix, nsgName string) (subnet *network.Subnet, err error) {
	nsg, err := a.GetNetworkSecurityGroup(ctx, nsgName)
	if err != nil {
		return subnet, fmt.Errorf("cannot get nsg: %v", err)
	}

	subnetsClient, err := a.getSubnetsClient()
	if err != nil {
		return nil, err
	}

	future, err := subnetsClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		subnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix:        to.StringPtr(addressPrefix),
				NetworkSecurityGroup: nsg,
			},
		})
	if err != nil {
		return subnet, fmt.Errorf("cannot create subnet: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, subnetsClient.Client)
	if err != nil {
		return subnet, fmt.Errorf("cannot get the subnet create or update future response: %v", err)
	}

	fmt.Printf("\nCreated subnet with security group")
	sc, err := future.Result(*subnetsClient)

	return &sc, err
}

// GetVirtualNetworkSubnet returns an existing subnet from a virtual
// network
func (a *Azure) GetVirtualNetworkSubnet(ctx context.Context, vnetName string, subnetName string) (subnet *network.Subnet, err error) {
	subnetsClient, err := a.getSubnetsClient()
	if err != nil {
		return
	}

	result, err := subnetsClient.Get(ctx, a.groupName, vnetName, subnetName, "")
	subnet = &result
	return
}

func (a *Azure) getNsgClient() (*network.SecurityGroupsClient, error) {
	nsgClient := network.NewSecurityGroupsClient(a.subID)
	authr, err := a.GetResourceManagementAuthorizer()
	if err != nil {
		return nil, err
	}
	nsgClient.Authorizer = authr
	nsgClient.AddToUserAgent(userAgent)
	return &nsgClient, nil
}

func (a Azure) buildFirewallRule(protocol network.SecurityRuleProtocol, port int) network.SecurityRule {
	portStr := strconv.Itoa(port)
	return network.SecurityRule{
		Name: to.StringPtr("allow_" + portStr),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Protocol:                 protocol,
			SourceAddressPrefix:      to.StringPtr("0.0.0.0/0"),
			SourcePortRange:          to.StringPtr("1-65535"),
			DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
			DestinationPortRange:     to.StringPtr(portStr),
			Access:                   network.SecurityRuleAccessAllow,
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 to.Int32Ptr(rand.Int31n(200-100) + 100), //Generating number between 100 - 200
		},
	}
}

// CreateNetworkSecurityGroup creates a new network security group with
// rules set for allowing SSH and HTTPS use
func (a *Azure) CreateNetworkSecurityGroup(ctx context.Context, location string, nsgName string, c *Config) (nsg *network.SecurityGroup, err error) {
	nsgClient, err := a.getNsgClient()
	if err != nil {
		return
	}

	var securityRules []network.SecurityRule

	for _, port := range c.RunConfig.Ports {
		var rule = a.buildFirewallRule(network.SecurityRuleProtocolTCP, port)
		securityRules = append(securityRules, rule)
	}

	for _, port := range c.RunConfig.UDPPorts {
		var rule = a.buildFirewallRule(network.SecurityRuleProtocolUDP, port)
		securityRules = append(securityRules, rule)
	}

	future, err := nsgClient.CreateOrUpdate(
		ctx,
		a.groupName,
		nsgName,
		network.SecurityGroup{
			Location: to.StringPtr(location),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: &securityRules,
			},
			Tags: getAzureDefaultTags(),
		},
	)

	if err != nil {
		return nsg, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nsgClient.Client)
	if err != nil {
		return nsg, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	nsgValue, err := future.Result(*nsgClient)
	nsg = &nsgValue
	return
}

// CreateSimpleNetworkSecurityGroup creates a new network security
// group, without rules (rules can be set later)
func (a *Azure) CreateSimpleNetworkSecurityGroup(ctx context.Context, location string, nsgName string) (nsg network.SecurityGroup, err error) {
	nsgClient, err := a.getNsgClient()
	if err != nil {
		return
	}

	future, err := nsgClient.CreateOrUpdate(
		ctx,
		a.groupName,
		nsgName,
		network.SecurityGroup{
			Location: to.StringPtr(location),
		},
	)

	if err != nil {
		return nsg, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nsgClient.Client)
	if err != nil {
		return nsg, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	nsg, err = future.Result(*nsgClient)
	return
}

// GetNetworkSecurityGroup returns an existing network security group
func (a *Azure) GetNetworkSecurityGroup(ctx context.Context, nsgName string) (sg *network.SecurityGroup, err error) {
	nsgClient, err := a.getNsgClient()
	if err != nil {
		return
	}

	result, err := nsgClient.Get(ctx, a.groupName, nsgName, "")
	sg = &result
	return
}
