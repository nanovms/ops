package lepton

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/olekukonko/tablewriter"
)

func formalizeAWSInstance(instance *ec2.Instance) *CloudInstance {
	instanceName := "unknown"
	for x := 0; x < len(instance.Tags); x++ {
		if aws.StringValue(instance.Tags[x].Key) == "Name" {
			instanceName = aws.StringValue(instance.Tags[x].Value)
		}
	}

	var privateIps, publicIps []string
	for _, ninterface := range instance.NetworkInterfaces {
		privateIps = append(privateIps, aws.StringValue(ninterface.PrivateIpAddress))

		if ninterface.Association != nil && ninterface.Association.PublicIp != nil {
			publicIps = append(publicIps, aws.StringValue(ninterface.Association.PublicIp))
		}
	}

	return &CloudInstance{
		ID:         aws.StringValue(instance.InstanceId),
		Name:       instanceName,
		Status:     aws.StringValue(instance.State.Name),
		Created:    aws.TimeValue(instance.LaunchTime).String(),
		PublicIps:  publicIps,
		PrivateIps: privateIps,
	}
}

func getAWSInstances(region string, filter []*ec2.Filter) []CloudInstance {
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	compute := ec2.New(svc)

	filter = append(filter, &ec2.Filter{Name: aws.String("tag:CreatedBy"), Values: aws.StringSlice([]string{"ops"})})

	request := ec2.DescribeInstancesInput{
		Filters: filter,
	}
	result, err := compute.DescribeInstances(&request)
	if err != nil {
		fmt.Println(err)
		exitWithError("failed getting instances")
	}

	var cinstances []CloudInstance

	for _, reservation := range result.Reservations {

		for i := 0; i < len(reservation.Instances); i++ {
			instance := reservation.Instances[i]

			cinstances = append(cinstances, *formalizeAWSInstance(instance))
		}

	}

	return cinstances
}

// StartInstance stops instance from AWS by ami name
func (p *AWS) StartInstance(ctx *Context, instanceID string) error {

	if instanceID == "" {
		exitWithError("Enter Instance ID")
	}

	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)

	compute := ec2.New(svc)

	if err != nil {
		exitWithError("Invalid region")
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	result, err := compute.StartInstances(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				exitWithError(aerr.Message())
			}
		} else {
			exitWithError(aerr.Message())
		}

	}

	if result.StartingInstances[0].InstanceId != nil {
		fmt.Printf("Started instance : %s\n", *result.StartingInstances[0].InstanceId)
	}

	return nil
}

// StopInstance stops instance from AWS by ami name
func (p *AWS) StopInstance(ctx *Context, instanceID string) error {

	if instanceID == "" {
		exitWithError("Enter InstanceID")
	}

	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)

	compute := ec2.New(svc)

	if err != nil {
		exitWithError("Invalid region")
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	result, err := compute.StopInstances(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				exitWithError(aerr.Message())
			}
		} else {
			exitWithError(aerr.Message())
		}

	}

	if result.StoppingInstances[0].InstanceId != nil {
		fmt.Printf("Stopped instance %s", *result.StoppingInstances[0].InstanceId)
	}

	return nil
}

// CreateInstance - Creates instance on AWS Platform
func (p *AWS) CreateInstance(ctx *Context) error {
	result, err := getAWSImages(p.ec2)
	if err != nil {
		exitWithError("Invalid zone")
	}

	imgName := ctx.config.CloudConfig.ImageName

	ami := ""
	var last time.Time
	layout := "2006-01-02T15:04:05.000Z"
	var image *ec2.Image

	for i := 0; i < len(result.Images); i++ {
		if result.Images[i].Tags != nil {
			for _, tag := range result.Images[i].Tags {
				if *tag.Key == "Name" && *tag.Value == imgName {
					image = result.Images[i]
					break
				}
			}
		}
	}

	if image == nil {
		return errors.New("can't find ami")
	}

	ami = aws.StringValue(image.ImageId)

	ntime := aws.StringValue(image.CreationDate)
	t, err := time.Parse(layout, ntime)
	if err != nil {
		return err
	}

	if last.Before(t) {
		last = t
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)

	// Create EC2 service client
	svc := ec2.New(sess)

	// create security group - could take a potential 'RemotePort' from
	// config.json in future
	vpc, err := p.GetVPC(ctx, svc)
	if err != nil {
		return err
	}

	if vpc == nil {
		vpc, err = p.CreateVPC(ctx, svc)
		if err != nil {
			return err
		}
	}

	var sg string

	if ctx.config.RunConfig.SecurityGroup != "" && ctx.config.RunConfig.VPC != "" {
		err = p.CheckValidSecurityGroup(ctx, svc)
		if err != nil {
			return err
		}

		sg = ctx.config.RunConfig.SecurityGroup
	} else {
		sg, err = p.CreateSG(ctx, svc, imgName, *vpc.VpcId)
		if err != nil {
			return err
		}
	}

	subnet, err := p.GetSubnet(ctx, svc, *vpc.VpcId)
	if err != nil {
		return err
	}

	if ctx.config.CloudConfig.Flavor == "" {
		ctx.config.CloudConfig.Flavor = "t2.micro"
	}

	// Create tags to assign to the instance
	tags, tagInstanceName := buildAwsTags(ctx.config.RunConfig.Tags, ctx.config.RunConfig.InstanceName)

	instanceInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: aws.String(ctx.config.CloudConfig.Flavor),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		SubnetId:     aws.String(*subnet.SubnetId),
		SecurityGroupIds: []*string{
			aws.String(sg),
		},
		TagSpecifications: []*ec2.TagSpecification{
			{ResourceType: aws.String("instance"), Tags: tags},
			{ResourceType: aws.String("volume"), Tags: tags},
		},
	}

	// Specify the details of the instance that you want to create.
	runResult, err := svc.RunInstances(instanceInput)

	if err != nil {
		fmt.Println("Could not create instance", err)
		return err
	}

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)

	// create dns zones/records to associate DNS record to instance IP
	if ctx.config.RunConfig.DomainName != "" {
		pollCount := 60
		for pollCount > 0 {
			fmt.Printf(".")
			time.Sleep(2 * time.Second)

			instance, err := p.GetInstanceByID(ctx, tagInstanceName)
			if err != nil {
				pollCount--
				continue
			}

			if len(instance.PublicIps) != 0 {
				err := CreateDNSRecord(ctx.config, instance.PublicIps[0], p)
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
func (p *AWS) GetInstances(ctx *Context) ([]CloudInstance, error) {
	cinstances := getAWSInstances(ctx.config.CloudConfig.Zone, nil)

	return cinstances, nil
}

// ListInstances lists instances on AWS
func (p *AWS) ListInstances(ctx *Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
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

		table.Append(rows)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from AWS
func (p *AWS) DeleteInstance(ctx *Context, instancename string) error {
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)
	compute := ec2.New(svc)

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instancename),
		},
	}

	_, err = compute.TerminateInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return err
	}

	// kill off any old security group as well

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *AWS) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (p *AWS) GetInstanceLogs(ctx *Context, instancename string) (string, error) {
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)
	compute := ec2.New(svc)

	// latest set to true is only avail on nitro (c5) instances
	// otherwise last 64k
	input := &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instancename),
	}

	result, err := compute.GetConsoleOutput(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(aws.StringValue(result.Output))
	if err != nil {
		return "", err
	}

	l := string(data)

	return l, nil
}
