//go:build aws || !onlyprovider

package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	smithy "github.com/aws/smithy-go"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/network"
	"github.com/nanovms/ops/types"
)

// GetSecurityGroup checks whether the configuration security group exists and has the configuration VPC assigned
func (p *AWS) GetSecurityGroup(execCtx context.Context, ctx *lepton.Context, svc *ec2.Client, vpc *awsEc2Types.Vpc) (sg *awsEc2Types.SecurityGroup, err error) {
	sgName := ctx.Config().CloudConfig.SecurityGroup

	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []awsEc2Types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: aws.ToStringSlice([]*string{aws.String(sgName)}),
			},
		},
	}

	var result *ec2.DescribeSecurityGroupsOutput

	result, err = svc.DescribeSecurityGroups(execCtx, input)
	if err != nil {
		return
	} else if len(result.SecurityGroups) == 0 {
		input := &ec2.DescribeSecurityGroupsInput{
			GroupIds: aws.ToStringSlice([]*string{aws.String(sgName)}),
		}

		result, err = svc.DescribeSecurityGroups(execCtx, input)
		if err != nil {
			err = fmt.Errorf("get security group with id '%s': %s", sg, err.Error())
			return
		}
	}

	if len(result.SecurityGroups) == 1 && *result.SecurityGroups[0].VpcId != *vpc.VpcId {
		err = fmt.Errorf("vpc mismatch: expected '%s' to have vpc '%s', got '%s'", sgName, *vpc.VpcId, *result.SecurityGroups[0].VpcId)
		return
	} else if len(result.SecurityGroups) == 0 {
		err = fmt.Errorf("security group '%s' not found", sgName)
		return
	}

	sg = &result.SecurityGroups[0]

	return
}

// GetSubnet returns a subnet with the context subnet name or the default subnet of vpc passed by argument
func (p *AWS) GetSubnet(execCtx context.Context, ctx *lepton.Context, svc *ec2.Client, vpcID string) (*awsEc2Types.Subnet, error) {
	subnetName := ctx.Config().CloudConfig.Subnet
	var filters []awsEc2Types.Filter
	var result *ec2.DescribeSubnetsOutput
	var err error

	filters = append(filters, awsEc2Types.Filter{Name: aws.String("vpc-id"), Values: []string{vpcID}})

	if subnetName != "" {
		subnetIDRegexp, _ := regexp.Compile("^subnet-.+")

		if subnetIDRegexp.Match([]byte(subnetName)) {
			result, err = svc.DescribeSubnets(execCtx, &ec2.DescribeSubnetsInput{
				SubnetIds: []string{subnetName},
				Filters:   filters,
			})
			if err != nil {
				err = fmt.Errorf("unable to describe subnets, %v", err)
				return nil, err
			}
		} else {
			result, err = svc.DescribeSubnets(execCtx, &ec2.DescribeSubnetsInput{
				Filters: append(filters, awsEc2Types.Filter{Name: aws.String("tag:Name"), Values: []string{subnetName}}),
			})
			if err != nil {
				err = fmt.Errorf("unable to describe subnets, %v", err)
				return nil, err
			}
		}

		if len(result.Subnets) != 0 {
			return &result.Subnets[0], nil
		}

	} else {
		input := &ec2.DescribeSubnetsInput{
			Filters: filters,
		}

		result, err = svc.DescribeSubnets(execCtx, input)
		if err != nil {
			err = fmt.Errorf("unable to describe subnets, %v", err)
			return nil, err
		}
	}

	if len(result.Subnets) == 0 && subnetName != "" {
		return nil, nil
	} else if len(result.Subnets) == 0 {
		return nil, nil
	}

	for _, subnet := range result.Subnets {
		if *subnet.DefaultForAz {
			return &subnet, nil
		}
	}

	return &result.Subnets[0], nil
}

// CreateSubnet creates a subnet on vpc
func (p *AWS) CreateSubnet(execCtx context.Context, ctx *lepton.Context, vpc *awsEc2Types.Vpc) (subnet *awsEc2Types.Subnet, err error) {
	tags, _ := buildAwsTags([]types.Tag{}, ctx.Config().CloudConfig.Subnet)

	createSubnetInput := &ec2.CreateSubnetInput{
		AvailabilityZone: aws.String(ctx.Config().CloudConfig.Zone),
		VpcId:            vpc.VpcId,
		CidrBlock:        vpc.CidrBlock,
		TagSpecifications: []awsEc2Types.TagSpecification{
			{Tags: tags, ResourceType: awsEc2Types.ResourceTypeSubnet},
		},
	}

	// set an ipv6 CIDR block in subnet if associated vpc has a ipv6 CIDR range
	if len(vpc.Ipv6CidrBlockAssociationSet) != 0 {
		ipv6Addr, _, err := net.ParseCIDR(*vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)
		if err != nil {
			return nil, err
		}
		createSubnetInput.Ipv6CidrBlock = types.StringPtr(ipv6Addr.String() + "/64")
	}

	result, err := p.ec2.CreateSubnet(execCtx, createSubnetInput)
	if err != nil {
		return
	}

	subnet = result.Subnet

	return
}

// GetVPC returns a vpc with the context vpc name or the default vpc
func (p *AWS) GetVPC(execCtx context.Context, ctx *lepton.Context, svc *ec2.Client) (*awsEc2Types.Vpc, error) {
	vpcName := ctx.Config().CloudConfig.VPC

	var vpc *awsEc2Types.Vpc
	var input *ec2.DescribeVpcsInput
	var result *ec2.DescribeVpcsOutput
	var err error

	if vpcName != "" {
		vpcIDRegexp, _ := regexp.Compile("^vpc-.+")

		if vpcIDRegexp.Match([]byte(vpcName)) {
			ctx.Logger().Debugf("no vpcs with name %s found", vpcName)
			ctx.Logger().Debugf("getting vpcs filtered by id %s", vpcName)
			input = &ec2.DescribeVpcsInput{
				VpcIds: []string{vpcName},
			}
			result, err = svc.DescribeVpcs(execCtx, input)
			if err != nil {
				return nil, fmt.Errorf("unable to describe VPCs, %v", err)
			}
		} else {
			ctx.Logger().Debugf("getting vpcs filtered by name %s", vpcName)
			var filters []awsEc2Types.Filter

			filters = append(filters, awsEc2Types.Filter{Name: aws.String("tag:Name"), Values: []string{vpcName}})
			input = &ec2.DescribeVpcsInput{
				Filters: filters,
			}

			result, err = svc.DescribeVpcs(execCtx, input)
			if err != nil {
				return nil, fmt.Errorf("unable to describe VPCs, %v", err)
			}
		}

		ctx.Logger().Debugf("found %d vpcs that match the criteria %s", len(result.Vpcs), vpcName)

		if len(result.Vpcs) != 0 {
			return &result.Vpcs[0], nil
		}
	} else {
		ctx.Logger().Debug("no vpc name specified")
		ctx.Logger().Debug("getting all vpcs")
		result, err = svc.DescribeVpcs(execCtx, input)
		if err != nil {
			return nil, fmt.Errorf("unable to describe VPCs, %v", err)
		}

		// select default vpc
		for i, s := range result.Vpcs {
			isDefault := *s.IsDefault
			if isDefault {
				ctx.Logger().Debug("picking default vpc")
				vpc = &result.Vpcs[i]
			}
		}

		// if there is no default VPC select the first vpc of the list
		if vpc == nil && len(result.Vpcs) != 0 {
			ctx.Logger().Debug("no default vpc found")
			vpc = &result.Vpcs[0]
			ctx.Logger().Debugf("picking vpc %+v", vpc)
		}
	}

	return vpc, nil
}

func (p *AWS) buildFirewallRule(protocol string, port string, ipv4, ipv6 bool) *awsEc2Types.IpPermission {
	fromPort := port
	toPort := port

	portsIntervalRegexp, _ := regexp.Compile(`\d+-\d+`)

	if match := portsIntervalRegexp.FindStringSubmatch(port); len(match) != 0 {
		rangeParts := strings.Split(port, "-")
		fromPort = rangeParts[0]
		toPort = rangeParts[1]
	}

	fromPortInt, err := strconv.Atoi(fromPort)
	if err != nil {
		fmt.Printf("Failed convert source port to integer, error is: %s", err)
		os.Exit(1)
	}

	toPortInt, err := strconv.Atoi(toPort)
	if err != nil {
		fmt.Printf("Failed convert destination port to integer, error is: %s", err)
		os.Exit(1)
	}

	var ec2Permission = new(awsEc2Types.IpPermission)
	ec2Permission.IpProtocol = &protocol
	ec2Permission.FromPort = aws.Int32(int32(fromPortInt))
	ec2Permission.ToPort = aws.Int32(int32(toPortInt))

	if ipv4 {
		ec2Permission.IpRanges = []awsEc2Types.IpRange{
			{CidrIp: aws.String("0.0.0.0/0")},
		}
	}

	if ipv6 {
		ec2Permission.Ipv6Ranges = []awsEc2Types.Ipv6Range{
			{CidrIpv6: aws.String("::/0")},
		}
	}

	return ec2Permission
}

// DeleteSG deletes a security group
func (p *AWS) DeleteSG(execCtx context.Context, groupID *string) {
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: groupID,
	}

	_, err := p.ec2.DeleteSecurityGroup(execCtx, input)
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

}

// CreateSG - Create security group
// instance specific
func (p *AWS) CreateSG(execCtx context.Context, ctx *lepton.Context, svc *ec2.Client, iname string, vpcID string) (sg *awsEc2Types.SecurityGroup, err error) {
	t := time.Now().UnixNano()
	s := strconv.FormatInt(t, 10)

	sgName := iname + "-" + s

	// Create tags to assign to the instance
	tags := []awsEc2Types.Tag{}
	tags = append(tags, awsEc2Types.Tag{Key: aws.String("ops-created"), Value: aws.String("true")})

	createRes, err := svc.CreateSecurityGroup(execCtx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String("security group for " + iname),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []awsEc2Types.TagSpecification{
			{ResourceType: awsEc2Types.ResourceTypeSecurityGroup, Tags: tags},
		},
	})
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			case "InvalidVpcID.NotFound":
				errstr := fmt.Sprintf("Unable to find VPC with ID %q.", vpcID)
				err = errors.New(errstr)
				return
			case "InvalidGroup.Duplicate":
				errstr := fmt.Sprintf("Security group %q already exists.", iname)
				err = errors.New(errstr)
				return
			}
		}
		errstr := fmt.Sprintf("Unable to create security group %q, %v", iname, err)
		err = errors.New(errstr)
		return
	}
	fmt.Printf("Created security group %s with VPC %s.\n", createRes.GroupId, vpcID)

	var ec2Permissions []awsEc2Types.IpPermission

	var ipv6 bool
	if ctx.Config().CloudConfig.EnableIPv6 {
		rule := p.buildFirewallRule("icmpv6", "-1", false, true)
		ec2Permissions = append(ec2Permissions, *rule)
		ipv6 = true
	}

	for _, port := range ctx.Config().RunConfig.Ports {
		rule := p.buildFirewallRule("tcp", port, true, ipv6)
		ec2Permissions = append(ec2Permissions, *rule)
	}

	for _, port := range ctx.Config().RunConfig.UDPPorts {
		rule := p.buildFirewallRule("udp", port, true, ipv6)
		ec2Permissions = append(ec2Permissions, *rule)
	}

	// maybe have these ports specified from config.json in near future
	if len(ec2Permissions) != 0 {
		_, err = svc.AuthorizeSecurityGroupIngress(execCtx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       createRes.GroupId,
			IpPermissions: ec2Permissions,
		})
		if err != nil {
			errstr := fmt.Sprintf("Unable to set security group %q ingress, %v", iname, err)
			err = errors.New(errstr)
			return
		}
	}

	result, err := svc.DescribeSecurityGroups(execCtx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{*createRes.GroupId},
	})
	if err != nil {
		return
	} else if len(result.SecurityGroups) == 0 {
		err = errors.New("failed creating security group")
	}

	sg = &result.SecurityGroups[0]

	return
}

// CreateVPC creates a virtual network
func (p *AWS) CreateVPC(execCtx context.Context, ctx *lepton.Context, svc *ec2.Client) (vpc *awsEc2Types.Vpc, err error) {
	vnetName := ctx.Config().CloudConfig.VPC

	if vnetName == "" {
		err = errors.New("specify vpc name")
		return
	}

	tags, _ := buildAwsTags([]types.Tag{}, vnetName)

	vpcs, err := svc.DescribeVpcs(execCtx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return
	}

	cidrBlocks := []string{}

	for _, v := range vpcs.Vpcs {
		cidrBlocks = append(cidrBlocks, *v.CidrBlock)
	}

	createInput := &ec2.CreateVpcInput{
		CidrBlock: aws.String(network.AllocateNewCidrBlock(cidrBlocks)),
		TagSpecifications: []awsEc2Types.TagSpecification{
			{Tags: tags, ResourceType: awsEc2Types.ResourceTypeVpc},
		},
	}

	if ctx.Config().CloudConfig.EnableIPv6 {
		createInput.AmazonProvidedIpv6CidrBlock = aws.Bool(true)
	}

	_, err = svc.CreateVpc(execCtx, createInput)
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				err = errors.New(aerr.Error())
			}
		} else {
			err = errors.New(err.Error())
		}
		return
	}

	vpc, err = p.GetVPC(execCtx, ctx, svc)
	if err != nil {
		return
	}

	// Add routes to allow external traffic
	var routeTable *awsEc2Types.RouteTable
	rtFilters := []awsEc2Types.Filter{{Name: aws.String("vpc-id"), Values: []string{*vpc.VpcId}}}
	describeRouteTablesOutput, err := svc.DescribeRouteTables(execCtx, &ec2.DescribeRouteTablesInput{Filters: rtFilters})
	if err != nil {
		return
	} else if len(describeRouteTablesOutput.RouteTables) != 0 {
		routeTable = &describeRouteTablesOutput.RouteTables[0]
	}

	if routeTable == nil {
		// it's unlikely that there is no routeTable
		// a route table is created and associated to the vpc when the vpc is created
		err = errors.New("vpc does not have any route table associated")
		return
	}

	var gw *awsEc2Types.InternetGateway
	gwFilters := []awsEc2Types.Filter{{Name: aws.String("attachment.vpc-id"), Values: []string{*vpc.VpcId}}}
	describeGatewaysOutput, err := svc.DescribeInternetGateways(execCtx, &ec2.DescribeInternetGatewaysInput{Filters: gwFilters})
	if err != nil {
		return
	} else if len(describeGatewaysOutput.InternetGateways) != 0 {
		gw = &describeGatewaysOutput.InternetGateways[0]
	}

	if gw == nil {
		gwInput := &ec2.CreateInternetGatewayInput{}
		gwOutput, err := svc.CreateInternetGateway(execCtx, gwInput)
		if err != nil {
			err = fmt.Errorf("failed creating an Internet Gateway: %v", err)
			return nil, err
		}

		_, err = svc.AttachInternetGateway(execCtx, &ec2.AttachInternetGatewayInput{VpcId: vpc.VpcId, InternetGatewayId: gwOutput.InternetGateway.InternetGatewayId})
		if err != nil {
			err = fmt.Errorf("failed attaching an Internet Gateway to a VPC: %v", err)
			return nil, err
		}

		gw = &awsEc2Types.InternetGateway{InternetGatewayId: gwOutput.InternetGateway.InternetGatewayId}
	}

	if ctx.Config().CloudConfig.EnableIPv6 {
		createRouteInput := &ec2.CreateRouteInput{
			DestinationIpv6CidrBlock: aws.String("::/0"),
			RouteTableId:             routeTable.RouteTableId,
			GatewayId:                gw.InternetGatewayId,
		}
		_, err = svc.CreateRoute(execCtx, createRouteInput)
		if err != nil {
			err = fmt.Errorf("failed creating ipv6 public route: %v", err)
			return
		}
	}

	createRouteInput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/24"),
		RouteTableId:         routeTable.RouteTableId,
		GatewayId:            gw.InternetGatewayId,
	}
	_, err = svc.CreateRoute(execCtx, createRouteInput)
	if err != nil {
		err = fmt.Errorf("failed creating ipv4 public route: %v", err)
		return
	}

	return
}
