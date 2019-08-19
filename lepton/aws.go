package lepton

import (
	"encoding/base64"
	"errors"

	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/olekukonko/tablewriter"
)

// AWS contains all operations for AWS
type AWS struct {
	Storage *S3
}

// BuildImage to be upload on AWS
func (p *AWS) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.customizeImage(ctx)
}

// BuildImageWithPackage to upload on AWS
func (p *AWS) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.customizeImage(ctx)
}

// Initialize AWS related things
func (p *AWS) Initialize() error {
	p.Storage = &S3{}
	return nil
}

// CreateImage - Creates image on AWS using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *AWS) CreateImage(ctx *Context) error {
	// this is a really convulted setup
	// 1) upload the image
	// 2) create a snapshot
	// 3) create an image

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)
	if err != nil {
		return err
	}

	compute := ec2.New(sess)

	c := ctx.config

	bucket := c.CloudConfig.BucketName
	key := c.CloudConfig.ImageName

	input := &ec2.ImportSnapshotInput{
		Description: aws.String("NanoVMs test"),
		DiskContainer: &ec2.SnapshotDiskContainer{
			Description: aws.String("NanoVMs test"),
			Format:      aws.String("raw"),
			UserBucket: &ec2.UserBucket{
				S3Bucket: aws.String(bucket),
				S3Key:    aws.String(key),
			},
		},
	}

	res, err := compute.ImportSnapshot(input)
	if err != nil {
		return err
	}

	taskFilter := &ec2.DescribeImportSnapshotTasksInput{
		ImportTaskIds: []*string{res.ImportTaskId},
	}

	describeOutput, err := compute.DescribeImportSnapshotTasks(taskFilter)
	if err != nil {
		return err
	}

	fmt.Println("waiting for snapshot - can take like 5min.... ")

	waitStartTime := time.Now()

	ct := aws.BackgroundContext()
	w := request.Waiter{
		Name:        "DescribeImportSnapshotTasks",
		Delay:       request.ConstantWaiterDelay(15 * time.Second),
		MaxAttempts: 60,
		Acceptors: []request.WaiterAcceptor{
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.PathAllWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "completed",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.PathAnyWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleted",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.PathAnyWaiterMatch,
				Argument: "ImportSnapshotTasks[].SnapshotTaskDetail.Status",
				Expected: "deleting",
			},
		},
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			req, _ := compute.DescribeImportSnapshotTasksRequest(taskFilter)
			req.SetContext(ct)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.WaitWithContext(ct)

	fmt.Printf("import done - took %f minutes\n", time.Since(waitStartTime).Minutes())

	// delete the tmp s3 image
	fmt.Println("deleting...")
	err = p.Storage.DeleteFromBucket(c, key)

	describeOutput, err = compute.DescribeImportSnapshotTasks(taskFilter)
	if err != nil {
		return err
	}

	fmt.Printf("snap: %v\n", describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId)

	snapshotID := describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId

	// tag the volume
	_, err = compute.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(key),
			},
		},
	})
	if err != nil {
		return err
	}

	// register ami
	rinput := &ec2.RegisterImageInput{
		Name:         aws.String(key),
		Architecture: aws.String("x86_64"),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(false),
					SnapshotId:          snapshotID,
					VolumeType:          aws.String("standard"),
				},
			},
		},
		Description:        aws.String(fmt.Sprintf("nanos image %s", key)),
		RootDeviceName:     aws.String("/dev/sda1"),
		VirtualizationType: aws.String("hvm"),
		EnaSupport:         aws.Bool(false),
	}

	resreg, err := compute.RegisterImage(rinput)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", resreg)

	return nil
}

func getAWSImages(region string) *ec2.DescribeImagesOutput {
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	compute := ec2.New(svc)

	input := &ec2.DescribeImagesInput{
		Owners: []*string{
			aws.String("self"),
		},
	}

	result, err := compute.DescribeImages(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	return result
}

func getAWSInstances(region string) *ec2.DescribeInstancesOutput {
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	compute := ec2.New(svc)

	request := ec2.DescribeInstancesInput{}
	result, err := compute.DescribeInstances(&request)
	if err != nil {
		fmt.Println(err)
	}

	return result
}

// ListImages lists images on AWS
func (p *AWS) ListImages(ctx *Context) error {
	result := getAWSImages(ctx.config.CloudConfig.Zone)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	images := result.Images
	for i := 0; i < len(images); i++ {
		var row []string
		row = append(row, aws.StringValue(images[i].Name))
		row = append(row, aws.StringValue(images[i].State))
		row = append(row, aws.StringValue(images[i].CreationDate))
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from AWS by ami name
func (p *AWS) DeleteImage(ctx *Context, imagename string) error {
	// delete ami by ami name
	svc, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)
	compute := ec2.New(svc)

	ec2Filters := []*ec2.Filter{}
	vals := []string{imagename}
	ec2Filters = append(ec2Filters, &ec2.Filter{Name: aws.String("name"), Values: aws.StringSlice(vals)})

	input := &ec2.DescribeImagesInput{
		Filters: ec2Filters,
	}

	result, err := compute.DescribeImages(input)
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

	amiID := aws.StringValue(result.Images[0].ImageId)
	snapID := aws.StringValue(result.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId)

	// grab snapshotid && grab image id

	params := &ec2.DeregisterImageInput{
		ImageId: aws.String(amiID),
		DryRun:  aws.Bool(false),
	}
	_, err = compute.DeregisterImage(params)
	if err != nil {
		return fmt.Errorf("Error running deregister image operation: %s", err)
	}

	// DeleteSnapshot

	params2 := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapID),
		DryRun:     aws.Bool(false),
	}
	_, err = compute.DeleteSnapshot(params2)
	if err != nil {
		return fmt.Errorf("Error running snapshot delete: %s", err)
	}

	return nil
}

// CreateInstance - Creates instance on AWS Platform
func (p *AWS) CreateInstance(ctx *Context) error {
	result := getAWSImages(ctx.config.CloudConfig.Zone)

	imgName := ctx.config.CloudConfig.ImageName

	ami := ""
	for i := 0; i < len(result.Images); i++ {
		n := aws.StringValue(result.Images[i].Name)
		if n == imgName {
			ami = aws.StringValue(result.Images[i].ImageId)
		}
	}

	if ami == "" {
		return errors.New("can't find ami")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(ctx.config.CloudConfig.Zone)},
	)

	// Create EC2 service client
	svc := ec2.New(sess)

	// Specify the details of the instance that you want to create.
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: aws.String("t2.micro"),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
	})

	if err != nil {
		fmt.Println("Could not create instance", err)
		return err
	}

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)

	// Add name tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(imgName),
			},
		},
	})
	if err != nil {
		fmt.Println("Could not create tags for instance", runResult.Instances[0].InstanceId, err)
		return err
	}

	return nil
}

// ListInstances lists instances on AWS
func (p *AWS) ListInstances(ctx *Context) error {
	result := getAWSInstances(ctx.config.CloudConfig.Zone)

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

	for _, reservation := range result.Reservations {

		for i := 0; i < len(reservation.Instances); i++ {
			instance := reservation.Instances[i]

			var rows []string

			instanceName := "unknown"
			for x := 0; x < len(instance.Tags); x++ {
				if aws.StringValue(instance.Tags[i].Key) == "Name" {
					instanceName = aws.StringValue(instance.Tags[i].Value)
				}
			}
			rows = append(rows, instanceName)
			rows = append(rows, aws.StringValue(instance.InstanceId))

			rows = append(rows, aws.StringValue(instance.State.Name))
			rows = append(rows, aws.TimeValue(instance.LaunchTime).String())

			var privateIps, publicIps []string
			for _, ninterface := range instance.NetworkInterfaces {
				privateIps = append(privateIps, aws.StringValue(ninterface.PrivateIpAddress))

				if ninterface.Association != nil && ninterface.Association.PublicIp != nil {
					publicIps = append(publicIps, aws.StringValue(ninterface.Association.PublicIp))
				}

			}
			rows = append(rows, strings.Join(privateIps, ","))
			rows = append(rows, strings.Join(publicIps, ","))
			table.Append(rows)
		}

	}

	table.Render()

	return nil
}

// DeleteInstance deletes instnace from AWS
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

	result, err := compute.TerminateInstances(input)
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

	fmt.Println(result)

	return nil
}

// GetInstanceLogs gets instance related logs
func (p *AWS) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {
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
		return err
	}

	data, err := base64.StdEncoding.DecodeString(aws.StringValue(result.Output))
	if err != nil {
		return err
	}

	l := string(data)
	fmt.Printf(l)

	return nil
}

// TODO - make me shared
func (p *AWS) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")
	return symlink, nil
}

// not an archive - just raw disk image
func (p *AWS) getArchiveName(ctx *Context) string {
	imagePath := ctx.config.RunConfig.Imagename
	return filepath.Join(filepath.Dir(imagePath), "disk.raw")
}
