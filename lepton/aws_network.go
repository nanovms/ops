package lepton

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nanovms/ops/config"
)

// CheckValidSecurityGroup checks whether the configuration security group exists and has the configuration VPC assigned
func (p *AWS) CheckValidSecurityGroup(ctx *Context, svc *ec2.EC2) error {
	sg := ctx.config.CloudConfig.SecurityGroup
	vpc := ctx.config.CloudConfig.VPC

	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			aws.String(sg),
		},
	}

	result, err := svc.DescribeSecurityGroups(input)
	if err != nil {
		return fmt.Errorf("get security group with id '%s': %s", sg, err.Error())
	}

	if len(result.SecurityGroups) == 1 && *result.SecurityGroups[0].VpcId != vpc {
		return fmt.Errorf("vpc mismatch: expected '%s' to have vpc '%s', got '%s'", sg, ctx.config.CloudConfig.VPC, *result.SecurityGroups[0].VpcId)
	} else if len(result.SecurityGroups) == 0 {
		return fmt.Errorf("security group '%s' not found", sg)
	}

	return nil
}

// GetSubnet returns a subnet with the context subnet name or the default subnet of vpc passed by argument
func (p *AWS) GetSubnet(ctx *Context, svc *ec2.EC2, vpcID string) (*ec2.Subnet, error) {
	subnetName := ctx.config.CloudConfig.Subnet
	var filters []*ec2.Filter

	filters = append(filters, &ec2.Filter{Name: aws.String("vpc-id"), Values: aws.StringSlice([]string{vpcID})})

	if subnetName != "" {
		filters = append(filters, &ec2.Filter{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{ctx.config.CloudConfig.Subnet})})
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: filters,
	}

	result, err := svc.DescribeSubnets(input)
	if err != nil {
		fmt.Printf("Unable to describe subnets, %v\n", err)
		return nil, err
	}

	if len(result.Subnets) == 0 && subnetName != "" {
		return nil, fmt.Errorf("No Subnets with name '%v' found to associate security group with", subnetName)
	} else if len(result.Subnets) == 0 {
		return nil, errors.New("No Subnets found to associate security group with")
	}

	if subnetName != "" {
		for _, subnet := range result.Subnets {
			if *subnet.DefaultForAz == true {
				return subnet, nil
			}
		}
	}

	return result.Subnets[0], nil
}

// GetVPC returns a vpc with the context vpc name or the default vpc
func (p *AWS) GetVPC(ctx *Context, svc *ec2.EC2) (*ec2.Vpc, error) {
	vpcName := ctx.config.CloudConfig.VPC
	var vpc *ec2.Vpc
	var input *ec2.DescribeVpcsInput
	if vpcName != "" {
		var filters []*ec2.Filter

		filters = append(filters, &ec2.Filter{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{ctx.config.CloudConfig.VPC})})
		input = &ec2.DescribeVpcsInput{
			Filters: filters,
		}
	}

	result, err := svc.DescribeVpcs(input)
	if err != nil {
		return nil, fmt.Errorf("Unable to describe VPCs, %v", err)
	}
	if len(result.Vpcs) == 0 && vpcName != "" {
		return nil, nil
	} else if len(result.Vpcs) == 0 {
		return nil, errors.New("No VPCs found to associate security group with")
	}

	if vpcName != "" {
		vpc = result.Vpcs[0]
	} else {
		// select default vpc
		for i, s := range result.Vpcs {
			isDefault := *s.IsDefault
			if isDefault == true {
				vpc = result.Vpcs[i]
			}
		}

		// if there is no default VPC select the first vpc of the list
		if vpc == nil {
			vpc = result.Vpcs[0]
		}
	}

	return vpc, nil
}

func (p AWS) buildFirewallRule(protocol string, port string) *ec2.IpPermission {
	fromPort := port
	toPort := port

	if strings.Contains(port, "-") {
		rangeParts := strings.Split(port, "-")
		fromPort = rangeParts[0]
		toPort = rangeParts[1]
	}

	fromPortInt, err := strconv.Atoi(fromPort)
	if err != nil {
		panic(err)
	}

	toPortInt, err := strconv.Atoi(toPort)
	if err != nil {
		panic(err)
	}

	var ec2Permission = new(ec2.IpPermission)
	ec2Permission.SetIpProtocol(protocol)
	ec2Permission.SetFromPort(int64(fromPortInt))
	ec2Permission.SetToPort(int64(toPortInt))
	ec2Permission.SetIpRanges([]*ec2.IpRange{
		{CidrIp: aws.String("0.0.0.0/0")},
	})

	return ec2Permission
}

// CreateSG - Create security group
func (p *AWS) CreateSG(ctx *Context, svc *ec2.EC2, imgName string, vpcID string) (string, error) {
	t := time.Now().UnixNano()
	s := strconv.FormatInt(t, 10)

	sgName := imgName + s

	createRes, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String("security group for " + imgName),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				errstr := fmt.Sprintf("Unable to find VPC with ID %q.", vpcID)
				return "", errors.New(errstr)
			case "InvalidGroup.Duplicate":
				errstr := fmt.Sprintf("Security group %q already exists.", imgName)
				return "", errors.New(errstr)
			}
		}
		errstr := fmt.Sprintf("Unable to create security group %q, %v", imgName, err)
		return "", errors.New(errstr)

	}
	fmt.Printf("Created security group %s with VPC %s.\n",
		aws.StringValue(createRes.GroupId), vpcID)

	var ec2Permissions []*ec2.IpPermission

	for _, port := range ctx.config.RunConfig.Ports {
		rule := p.buildFirewallRule("tcp", port)
		ec2Permissions = append(ec2Permissions, rule)
	}

	for _, port := range ctx.config.RunConfig.UDPPorts {
		rule := p.buildFirewallRule("udp", port)
		ec2Permissions = append(ec2Permissions, rule)
	}

	// maybe have these ports specified from config.json in near future
	if len(ec2Permissions) != 0 {
		_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       createRes.GroupId,
			IpPermissions: ec2Permissions,
		})
		if err != nil {
			errstr := fmt.Sprintf("Unable to set security group %q ingress, %v", imgName, err)
			return "", errors.New(errstr)
		}
	}

	return aws.StringValue(createRes.GroupId), nil
}

// CreateVPC creates a virtual network
func (p *AWS) CreateVPC(ctx *Context, svc *ec2.EC2) (vpc *ec2.Vpc, err error) {
	vnetName := ctx.config.CloudConfig.VPC

	if vnetName == "" {
		err = errors.New("specify vpc name")
		return
	}

	tags, _ := buildAwsTags([]config.Tag{}, vnetName)

	vpcs, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return
	}

	cidrBlocks := []string{}

	for _, v := range vpcs.Vpcs {
		cidrBlocks = append(cidrBlocks, *v.CidrBlock)
	}

	_, err = svc.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String(AllocateNewCidrBlock(cidrBlocks)),
		TagSpecifications: []*ec2.TagSpecification{
			{Tags: tags, ResourceType: aws.String("vpc")},
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				err = errors.New(aerr.Error())
			}
		} else {
			err = errors.New(err.Error())
		}
		return
	}

	vpc, err = p.GetVPC(ctx, svc)

	if err == nil {
		tags, _ = buildAwsTags([]config.Tag{}, ctx.config.CloudConfig.Subnet)

		_, err = svc.CreateSubnet(&ec2.CreateSubnetInput{
			VpcId:     vpc.VpcId,
			CidrBlock: vpc.CidrBlock,
			TagSpecifications: []*ec2.TagSpecification{
				{Tags: tags, ResourceType: aws.String("subnet")},
			},
		})
	}

	return
}
