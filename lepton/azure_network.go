package lepton

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func (a *Azure) getNicClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(a.subID)
	auth, _ := a.GetResourceManagementAuthorizer()
	nicClient.Authorizer = auth
	nicClient.AddToUserAgent(userAgent)
	return nicClient
}

// CreateNIC creates a new network interface. The Network Security Group
// is not a required parameter
func (a *Azure) CreateNIC(ctx context.Context, vnetName, subnetName, nsgName, ipName, nicName string) (nic network.Interface, err error) {
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
		Location: to.StringPtr(a.locationDefault),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &ip,
					},
				},
			},
		},
	}

	if nsgName != "" {
		nsg, err := a.GetNetworkSecurityGroup(ctx, nsgName)
		if err != nil {
			log.Fatalf("failed to get nsg: %v", err)
		}
		nicParams.NetworkSecurityGroup = &nsg
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

	return future.Result(nicClient)
}

func (a *Azure) getIPClient() network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClient(a.subID)
	auth, _ := a.GetResourceManagementAuthorizer()
	ipClient.Authorizer = auth
	ipClient.AddToUserAgent(userAgent)
	return ipClient
}

// CreatePublicIP creates a new public IP
func (a *Azure) CreatePublicIP(ctx context.Context, ipName string) (ip network.PublicIPAddress, err error) {
	ipClient := a.getIPClient()
	future, err := ipClient.CreateOrUpdate(
		ctx,
		a.groupName,
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(a.locationDefault),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)

	if err != nil {
		return ip, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		return ip, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	return future.Result(ipClient)
}

// GetPublicIP returns an existing public IP
func (a *Azure) GetPublicIP(ctx context.Context, ipName string) (network.PublicIPAddress, error) {
	ipClient := a.getIPClient()
	return ipClient.Get(ctx, a.groupName, ipName, "")
}

func (a *Azure) getVnetClient() network.VirtualNetworksClient {
	vnetClient := network.NewVirtualNetworksClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	vnetClient.Authorizer = authr
	vnetClient.AddToUserAgent(userAgent)
	return vnetClient
}

// CreateVirtualNetwork creates a virtual network
func (a *Azure) CreateVirtualNetwork(ctx context.Context, vnetName string) (vnet network.VirtualNetwork, err error) {
	vnetClient := a.getVnetClient()
	future, err := vnetClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(a.locationDefault),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
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

	return future.Result(vnetClient)
}

// CreateVirtualNetworkAndSubnets creates a virtual network with two
// subnets
func (a *Azure) CreateVirtualNetworkAndSubnets(ctx context.Context, vnetName, subnet1Name, subnet2Name string) (vnet network.VirtualNetwork, err error) {
	vnetClient := a.getVnetClient()
	future, err := vnetClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(a.locationDefault),
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

	return future.Result(vnetClient)
}

func (a *Azure) getSubnetsClient() network.SubnetsClient {
	subnetsClient := network.NewSubnetsClient(a.subID)
	auth, _ := a.GetResourceManagementAuthorizer()
	subnetsClient.Authorizer = auth
	subnetsClient.AddToUserAgent(userAgent)
	return subnetsClient
}

// CreateVirtualNetworkSubnet creates a subnet in an existing vnet
func (a *Azure) CreateVirtualNetworkSubnet(ctx context.Context, vnetName, subnetName string) (subnet network.Subnet, err error) {
	subnetsClient := a.getSubnetsClient()

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

	return future.Result(subnetsClient)
}

// CreateSubnetWithNetworkSecurityGroup create a subnet referencing a
// network security group
func (a *Azure) CreateSubnetWithNetworkSecurityGroup(ctx context.Context, vnetName, subnetName, addressPrefix, nsgName string) (subnet network.Subnet, err error) {
	nsg, err := a.GetNetworkSecurityGroup(ctx, nsgName)
	if err != nil {
		return subnet, fmt.Errorf("cannot get nsg: %v", err)
	}

	subnetsClient := a.getSubnetsClient()
	future, err := subnetsClient.CreateOrUpdate(
		ctx,
		a.groupName,
		vnetName,
		subnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix:        to.StringPtr(addressPrefix),
				NetworkSecurityGroup: &nsg,
			},
		})
	if err != nil {
		return subnet, fmt.Errorf("cannot create subnet: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, subnetsClient.Client)
	if err != nil {
		return subnet, fmt.Errorf("cannot get the subnet create or update future response: %v", err)
	}

	return future.Result(subnetsClient)
}

// GetVirtualNetworkSubnet returns an existing subnet from a virtual
// network
func (a *Azure) GetVirtualNetworkSubnet(ctx context.Context, vnetName string, subnetName string) (network.Subnet, error) {
	subnetsClient := a.getSubnetsClient()
	return subnetsClient.Get(ctx, a.groupName, vnetName, subnetName, "")
}

func (a *Azure) getNsgClient() network.SecurityGroupsClient {
	nsgClient := network.NewSecurityGroupsClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	nsgClient.Authorizer = authr
	nsgClient.AddToUserAgent(userAgent)
	return nsgClient
}

// CreateNetworkSecurityGroup creates a new network security group with
// rules set for allowing SSH and HTTPS use
func (a *Azure) CreateNetworkSecurityGroup(ctx context.Context, nsgName string) (nsg network.SecurityGroup, err error) {
	nsgClient := a.getNsgClient()
	future, err := nsgClient.CreateOrUpdate(
		ctx,
		a.groupName,
		nsgName,
		network.SecurityGroup{
			Location: to.StringPtr(a.locationDefault),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: &[]network.SecurityRule{
					{
						Name: to.StringPtr("allow_https"),
						SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
							Protocol:                 network.SecurityRuleProtocolTCP,
							SourceAddressPrefix:      to.StringPtr("0.0.0.0/0"),
							SourcePortRange:          to.StringPtr("1-65535"),
							DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
							DestinationPortRange:     to.StringPtr("443"),
							Access:                   network.SecurityRuleAccessAllow,
							Direction:                network.SecurityRuleDirectionInbound,
							Priority:                 to.Int32Ptr(200),
						},
					},
				},
			},
		},
	)

	if err != nil {
		return nsg, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nsgClient.Client)
	if err != nil {
		return nsg, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	return future.Result(nsgClient)
}

// CreateSimpleNetworkSecurityGroup creates a new network security
// group, without rules (rules can be set later)
func (a *Azure) CreateSimpleNetworkSecurityGroup(ctx context.Context, nsgName string) (nsg network.SecurityGroup, err error) {
	nsgClient := a.getNsgClient()
	future, err := nsgClient.CreateOrUpdate(
		ctx,
		a.groupName,
		nsgName,
		network.SecurityGroup{
			Location: to.StringPtr(a.locationDefault),
		},
	)

	if err != nil {
		return nsg, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, nsgClient.Client)
	if err != nil {
		return nsg, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	return future.Result(nsgClient)
}

// GetNetworkSecurityGroup returns an existing network security group
func (a *Azure) GetNetworkSecurityGroup(ctx context.Context, nsgName string) (network.SecurityGroup, error) {
	nsgClient := a.getNsgClient()
	return nsgClient.Get(ctx, a.groupName, nsgName, "")
}
