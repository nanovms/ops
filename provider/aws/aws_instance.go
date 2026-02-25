//go:build aws || !onlyprovider

package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awsEc2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	smithy "github.com/aws/smithy-go"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

func formalizeAWSInstance(instance *awsEc2Types.Instance) *lepton.CloudInstance {
	imageName := "unknown"
	instanceName := "unknown"
	for x := 0; x < len(instance.Tags); x++ {
		if aws.ToString(instance.Tags[x].Key) == "Name" {
			instanceName = aws.ToString(instance.Tags[x].Value)
		} else if aws.ToString(instance.Tags[x].Key) == "image" {
			imageName = aws.ToString(instance.Tags[x].Value)
		}
	}

	var privateIps, publicIps []string
	for _, ninterface := range instance.NetworkInterfaces {
		privateIps = append(privateIps, aws.ToString(ninterface.PrivateIpAddress))

		if ninterface.Association != nil && ninterface.Association.PublicIp != nil {
			publicIps = append(publicIps, aws.ToString(ninterface.Association.PublicIp))
		}
	}

	return &lepton.CloudInstance{
		ID:         aws.ToString(instance.InstanceId),
		Name:       instanceName,
		Status:     string(instance.State.Name),
		Created:    aws.ToTime(instance.LaunchTime).String(),
		PublicIps:  publicIps,
		PrivateIps: privateIps,
		Image:      imageName,
	}
}

func getAWSInstances(execCtx context.Context, ec2Client *ec2.Client, region string, filter []awsEc2Types.Filter) ([]lepton.CloudInstance, error) {
	filter = append(filter, awsEc2Types.Filter{Name: aws.String("tag:CreatedBy"), Values: []string{"ops"}})

	request := ec2.DescribeInstancesInput{
		Filters: filter,
	}
	result, err := ec2Client.DescribeInstances(execCtx, &request, func(opts *ec2.Options) { opts.Region = stripZone(region) })
	if err != nil {
		log.Fatalf("failed getting instances: ", err.Error())
	}

	var cinstances []lepton.CloudInstance

	for _, reservation := range result.Reservations {

		for i := 0; i < len(reservation.Instances); i++ {
			instance := reservation.Instances[i]

			cinstances = append(cinstances, *formalizeAWSInstance(&instance))
		}

	}

	return cinstances, nil
}

// RebootInstance reboots the instance.
func (p *AWS) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance stops instance from AWS by ami name
func (p *AWS) StartInstance(ctx *lepton.Context, instanceName string) error {

	if instanceName == "" {
		return errors.New("enter instance name")
	}

	instance, err := p.findInstanceByName(instanceName)
	if err != nil {
		return err
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{
			aws.ToString(instance.InstanceId),
		},
	}

	result, err := p.ec2.StartInstances(p.execCtx, input)

	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				return errors.New(aerr.ErrorMessage())
			}
		} else {
			return errors.New(aerr.ErrorMessage())
		}

	}

	if result.StartingInstances[0].InstanceId != nil {
		fmt.Printf("Started instance : %s\n", *result.StartingInstances[0].InstanceId)
	}

	return nil
}

// StopInstance stops instance from AWS by ami name
func (p *AWS) StopInstance(ctx *lepton.Context, instanceName string) error {
	if instanceName == "" {
		return errors.New("enter instance name")
	}

	instance, err := p.findInstanceByName(instanceName)
	if err != nil {
		return err
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	}

	result, err := p.ec2.StopInstances(p.execCtx, input)

	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				return errors.New(aerr.ErrorMessage())
			}
		} else {
			return errors.New(aerr.ErrorMessage())
		}

	}

	if result.StoppingInstances[0].InstanceId != nil {
		fmt.Printf("Stopped instance %s\n", *result.StoppingInstances[0].InstanceId)
	}

	return nil
}

// CreateInstance - Creates instance on AWS Platform
func (p *AWS) CreateInstance(ctx *lepton.Context) error {
	ctx.Logger().Debug("getting aws images")
	result, err := getAWSImages(p.execCtx, p.ec2)
	if err != nil {
		ctx.Logger().Errorf("failed getting images")
		return err
	}

	imgName := ctx.Config().CloudConfig.ImageName

	ami := ""
	var last time.Time
	layout := "2006-01-02T15:04:05.000Z"
	var image *awsEc2Types.Image

	rv := ctx.Config().CloudConfig.RootVolume

	// if custom block device settings are set we need the snapshotid
	snapID := ""
	for i := 0; i < len(result.Images); i++ {
		if result.Images[i].Tags != nil {
			for _, tag := range result.Images[i].Tags {
				if *tag.Key == "Name" && *tag.Value == imgName {
					image = &result.Images[i]
					snapID = *(result.Images[i].BlockDeviceMappings[0].Ebs.SnapshotId)
					break
				}
			}
		}
	}

	if image == nil {
		return fmt.Errorf("can't find ami with name %s", imgName)
	}

	ami = aws.ToString(image.ImageId)

	ntime := aws.ToString(image.CreationDate)
	t, err := time.Parse(layout, ntime)
	if err != nil {
		return err
	}

	if last.Before(t) {
		last = t
	}

	svc := p.ec2
	cloudConfig := ctx.Config().CloudConfig
	runConfig := ctx.Config().RunConfig

	// create security group - could take a potential 'RemotePort' from
	// config.json in future
	ctx.Logger().Debug("getting vpc")
	vpc, err := p.GetVPC(p.execCtx, ctx, svc)
	if err != nil {
		return err
	}

	if vpc == nil {
		ctx.Logger().Debugf("creating vpc with name %s", cloudConfig.VPC)
		vpc, err = p.CreateVPC(p.execCtx, ctx, svc)
		if err != nil {
			return err
		}
	}

	var sg *awsEc2Types.SecurityGroup

	if cloudConfig.SecurityGroup != "" && cloudConfig.VPC != "" {
		ctx.Logger().Debugf("getting security group with name %s", cloudConfig.SecurityGroup)
		sg, err = p.GetSecurityGroup(p.execCtx, ctx, svc, vpc)
		if err != nil {
			return err
		}
	} else {
		iname := ctx.Config().RunConfig.InstanceName
		ctx.Logger().Debugf("creating new security group in vpc %s", *vpc.VpcId)
		sg, err = p.CreateSG(p.execCtx, ctx, svc, iname, *vpc.VpcId)
		if err != nil {
			return err
		}
	}

	ctx.Logger().Debug("getting subnet")
	var subnet *awsEc2Types.Subnet
	subnet, err = p.GetSubnet(p.execCtx, ctx, svc, *vpc.VpcId)
	if err != nil {
		return err
	}

	if subnet == nil {
		subnet, err = p.CreateSubnet(p.execCtx, ctx, vpc)
		if err != nil {
			return err
		}
	}

	if cloudConfig.Flavor == "" {
		cloudConfig.Flavor = "t2.micro"
	}

	// Create tags to assign to the instance
	tags, tagInstanceName := buildAwsTags(cloudConfig.Tags, ctx.Config().RunConfig.InstanceName)
	tags = append(tags, awsEc2Types.Tag{Key: aws.String("image"), Value: &imgName})

	instanceNIS := &awsEc2Types.InstanceNetworkInterfaceSpecification{
		DeleteOnTermination: aws.Bool(true),
		DeviceIndex:         aws.Int32(0),
		Groups: []string{
			aws.ToString(sg.GroupId),
		},
		SubnetId: aws.String(*subnet.SubnetId),
	}

	instanceInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: awsEc2Types.InstanceType(cloudConfig.Flavor),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		TagSpecifications: []awsEc2Types.TagSpecification{
			{ResourceType: awsEc2Types.ResourceType("instance"), Tags: tags},
			{ResourceType: awsEc2Types.ResourceType("volume"), Tags: tags},
		},
	}

	if rv.IsCustom() {
		ctx.Logger().Debug("setting custom root settings")
		ebs := &awsEc2Types.EbsBlockDevice{}
		ebs.SnapshotId = aws.String(snapID)

		if rv.Typeof != "" {
			ebs.VolumeType = awsEc2Types.VolumeType(rv.Typeof)
		}

		if rv.Iops != 0 {
			if rv.Typeof == "" {
				fmt.Println("Setting iops is not supported for gp2")
				os.Exit(1)
			}

			ebs.Iops = aws.Int32(int32(rv.Iops))
		}

		if rv.Throughput != 0 {
			if rv.Typeof == "" {
				fmt.Println("You can not provision iops without setting type to gp3")
				os.Exit(1)
			}

			ebs.Throughput = aws.Int32(int32(rv.Throughput))
		}

		if rv.Size != 0 {
			ebs.VolumeSize = aws.Int32(int32(rv.Size))
		}

		instanceInput.BlockDeviceMappings = []awsEc2Types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs:        ebs,
			},
		}
	}

	if ctx.Config().CloudConfig.DedicatedHostID != "" {
		instanceInput.Placement = &awsEc2Types.Placement{
			HostId: aws.String(ctx.Config().CloudConfig.DedicatedHostID),
		}
	}

	if ctx.Config().RunConfig.IPAddress != "" {
		instanceNIS.PrivateIpAddress = aws.String(ctx.Config().RunConfig.IPAddress)
	}

	if ctx.Config().CloudConfig.EnableIPv6 {
		if ctx.Config().RunConfig.IPv6Address != "" {
			v6ad := ctx.Config().RunConfig.IPv6Address
			addie := &awsEc2Types.InstanceIpv6Address{
				Ipv6Address: aws.String(v6ad),
			}
			instanceNIS.Ipv6Addresses = []awsEc2Types.InstanceIpv6Address{*addie}
		} else {
			instanceNIS.Ipv6AddressCount = aws.Int32(1)
		}
	}

	instanceInput.NetworkInterfaces = []awsEc2Types.InstanceNetworkInterfaceSpecification{*instanceNIS}

	if cloudConfig.InstanceProfile != "" {
		instanceInput.IamInstanceProfile = &awsEc2Types.IamInstanceProfileSpecification{
			Name: aws.String(cloudConfig.InstanceProfile),
		}
	}

	if runConfig.InstanceGroup != "" {
		ltInput := &LaunchTemplateInput{
			AutoScalingGroup:    runConfig.InstanceGroup,
			InstanceProfileName: cloudConfig.InstanceProfile,
			LaunchTemplateName:  tagInstanceName,
			ImageID:             ami,
			InstanceType:        cloudConfig.Flavor,
			Tags:                tags,
		}
		if err := p.autoScalingLaunchTemplate(ltInput); err != nil {
			log.Errorf("Could not create launch template for auto scaling group %v", err)
			return err
		}

		instanceInput.LaunchTemplate = &awsEc2Types.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(ltInput.LaunchTemplateName),
			Version:            aws.String("$Default"),
		}
	}

	// Add user data if provided
	if cloudConfig.UserData != "" {
		userData := lepton.EncodeUserDataBase64(cloudConfig.UserData)
		instanceInput.UserData = aws.String(userData)
	}

	// Specify the details of the instance that you want to create.
	ctx.Logger().Debugf("running instance with input %v", instanceInput)
	_, err = svc.RunInstances(p.execCtx, instanceInput)
	if err != nil {
		log.Errorf("Could not create instance %v", err)
		return err
	}

	log.Info("Created instance", tagInstanceName)

	if cloudConfig.StaticIP != "" {
		log.Debug("associating elastic IP to instance")
		var instance *lepton.CloudInstance
		pollCount := 60
		for pollCount > 0 {
			time.Sleep(2 * time.Second)
			instance, err = p.GetInstanceByName(ctx, tagInstanceName)
			if err == nil {
				pollCount--
				if instance.Status != "pending" {
					break
				}
			} else {
				break
			}
		}
		if err == nil {
			input := &ec2.AssociateAddressInput{
				InstanceId: aws.String(instance.ID),
				PublicIp:   aws.String(cloudConfig.StaticIP),
			}
			result, err := svc.AssociateAddress(p.execCtx, input)
			if err != nil {
				log.Errorf("Could not associate elastic IP: %v", err.(smithy.APIError).Error())
			} else {
				log.Debugf("result: %v", result)
			}
		} else {
			log.Errorf("Could not retrieve instance: %v", err.Error())
		}
	}

	// create dns zones/records to associate DNS record to instance IP
	if cloudConfig.DomainName != "" {
		ctx.Logger().Debug("associating domain to the created instance")
		pollCount := 60
		for pollCount > 0 {
			fmt.Printf(".")
			time.Sleep(2 * time.Second)

			ctx.Logger().Debug("getting instance")
			instance, err := p.GetInstanceByName(ctx, tagInstanceName)
			if err != nil {
				pollCount--
				continue
			}

			if len(instance.PublicIps) != 0 {
				ctx.Logger().Debugf("creating dns record %s with ip %s", cloudConfig.DomainName, instance.PublicIps[0])
				err := lepton.CreateDNSRecord(ctx.Config(), instance.PublicIps[0], p)
				if err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("\nOperation timed out. No of tries %d", pollCount)
	}

	return nil
}

// GetInstances return all instances on AWS
func (p *AWS) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	cinstances, err := getAWSInstances(p.execCtx, p.ec2, ctx.Config().CloudConfig.Zone, nil)
	if err != nil {
		return nil, err
	}
	return cinstances, nil
}

// GetInstanceByName returns instance with given name
func (p *AWS) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	var filters []awsEc2Types.Filter
	filters = append(filters, awsEc2Types.Filter{Name: aws.String("tag:Name"), Values: []string{name}})
	instances, err := getAWSInstances(p.execCtx, p.ec2, ctx.Config().CloudConfig.Zone, filters)
	if err != nil {
		return nil, err
	} else if len(instances) == 0 {
		return nil, lepton.ErrInstanceNotFound(name)
	}
	return &instances[0], nil
}

// ListInstances lists instances on AWS
func (p *AWS) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(instances)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created", "Private Ips", "Public Ips", "Image"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range instances {

		var rows []string

		rows = append(rows, instance.Name)
		rows = append(rows, instance.ID)

		rows = append(rows, instance.Status)
		rows = append(rows, instance.Created)

		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))

		rows = append(rows, instance.Image)

		table.Append(rows)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from AWS
// if deleting sgs created by ops it can take a while so might be worth
// optionally making this go into the background
func (p *AWS) DeleteInstance(ctx *lepton.Context, instanceName string) error {
	if instanceName == "" {
		return errors.New("enter instance name")
	}

	instance, err := p.findInstanceByName(instanceName)
	if err != nil {
		return err
	}

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{
			aws.ToString(instance.InstanceId),
		},
	}

	sg, err := p.findSGByName(instanceName)
	if err != nil {
		return err
	}

	_, err = p.ec2.TerminateInstances(p.execCtx, input)
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				log.Error(aerr)
			}
		} else {
			log.Error(err)
		}
		return err
	}

	if sg != nil {
		fmt.Println("waiting for sg to be removed")

		i2 := &ec2.DescribeInstancesInput{
			InstanceIds: []string{
				aws.ToString(instance.InstanceId),
			},
		}

		_, err = WaitUntilEc2InstanceTerminated(p.execCtx, p.ec2, i2)
		if err != nil {
			fmt.Println(err)
		}

		p.DeleteSG(p.execCtx, sg.GroupId)
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *AWS) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Println(l)
	return nil
}

// InstanceStats show metrics for instances on aws.
func (p *AWS) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not avilable")
}

// GetInstanceLogs gets instance related logs
func (p *AWS) GetInstanceLogs(ctx *lepton.Context, instanceName string) (string, error) {
	if instanceName == "" {
		return "", errors.New("enter instance name")
	}

	instance, err := p.findInstanceByName(instanceName)
	if err != nil {
		return "", err
	}

	// latest set to true is only avail on nitro (c5) instances
	// otherwise last 64k
	input := &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(*instance.InstanceId),
	}

	result, err := p.ec2.GetConsoleOutput(p.execCtx, input)
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
			default:
				log.Error(aerr)
			}
		} else {
			log.Error(err)
		}
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(aws.ToString(result.Output))
	if err != nil {
		return "", err
	}

	l := string(data)

	return l, nil
}

func (p *AWS) findInstanceByName(name string) (*awsEc2Types.Instance, error) {
	filter := []awsEc2Types.Filter{
		{Name: aws.String("tag:CreatedBy"), Values: []string{"ops"}},
		{Name: aws.String("tag:Name"), Values: []string{name}},
		{Name: aws.String("instance-state-name"), Values: []string{"running", "pending", "shutting-down", "stopping", "stopped"}},
	}

	request := ec2.DescribeInstancesInput{
		Filters: filter,
	}
	result, err := p.ec2.DescribeInstances(p.execCtx, &request)
	if err != nil {
		return nil, fmt.Errorf("failed getting instances: %v", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance with name %s not found", name)
	}

	return &result.Reservations[0].Instances[0], nil
}

// bit of a hack here
// can convert to explicit tag
// currently only returns sgs created by ops
func (p *AWS) findSGByName(name string) (*awsEc2Types.SecurityGroup, error) {
	filter := []awsEc2Types.Filter{
		{Name: aws.String("tag:ops-created"), Values: []string{"true"}},
		{Name: aws.String("description"), Values: []string{"security group for " + name}},
	}

	request := ec2.DescribeSecurityGroupsInput{
		Filters: filter,
	}
	result, err := p.ec2.DescribeSecurityGroups(p.execCtx, &request)
	if err != nil {
		return nil, fmt.Errorf("failed getting security group: %v", err)
	}

	return &result.SecurityGroups[0], nil
}
