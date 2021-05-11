package aws

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"

	"github.com/schollz/progressbar/v3"
)

// BuildImage to be upload on AWS
func (p *AWS) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on AWS
func (p *AWS) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.CustomizeImage(ctx)
}

// CreateImage - Creates image on AWS using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *AWS) CreateImage(ctx *lepton.Context, imagePath string) error {
	// this is a really convulted setup
	// 1) upload the image
	// 2) create a snapshot
	// 3) create an image
	imageName := ctx.Config().CloudConfig.ImageName

	i, _ := p.findImageByName(imageName)
	if i != nil {
		return fmt.Errorf("failed creating image: image with name %s already exists", imageName)
	}

	err := p.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	c := ctx.Config()

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

	ctx.Logger().Info("Importing snapshot from s3 image file")
	res, err := p.ec2.ImportSnapshot(input)
	if err != nil {
		return err
	}

	snapshotID, err := p.waitSnapshotToBeReady(c, res.ImportTaskId)
	if err != nil {
		return err
	}

	// delete the tmp s3 image
	ctx.Logger().Info("Deleting s3 image file")
	err = p.Storage.DeleteFromBucket(c, key)
	if err != nil {
		return err
	}

	// tag the volume
	tags, _ := buildAwsTags(c.CloudConfig.Tags, key)

	ctx.Logger().Log("Tagging snapshot")
	_, err = p.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{snapshotID},
		Tags:      tags,
	})
	if err != nil {
		return err
	}

	t := time.Now().UnixNano()
	s := strconv.FormatInt(t, 10)

	amiName := key + s

	// register ami
	enaSupport := GetEnaSupportForFlavor(c.CloudConfig.Flavor)

	rinput := &ec2.RegisterImageInput{
		Name:         aws.String(amiName),
		Architecture: aws.String("x86_64"),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(false),
					SnapshotId:          snapshotID,
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		Description:        aws.String(fmt.Sprintf("nanos image %s", key)),
		RootDeviceName:     aws.String("/dev/sda1"),
		VirtualizationType: aws.String("hvm"),
		EnaSupport:         aws.Bool(enaSupport),
	}

	ctx.Logger().Info("Registering image")
	resreg, err := p.ec2.RegisterImage(rinput)
	if err != nil {
		return err
	}

	// Add name tag to the created ami
	ctx.Logger().Info("Tagging image")
	_, err = p.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{resreg.ImageId},
		Tags:      tags,
	})

	return nil
}

var (
	// NitroInstanceTypes are the AWS virtualized types built on the Nitro system.
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html#ec2-nitro-instances
	NitroInstanceTypes = map[string]bool{
		"a1":   true,
		"c5":   true,
		"c5a":  true,
		"c5ad": true,
		"c5d":  true,
		"c5n":  true,
		"c6g":  true,
		"c6gd": true,
		"c6gn": true,
		"d3":   true,
		"d3en": true,
		"g4":   true,
		"i3en": true,
		"inf1": true,
		"m5":   true,
		"m5a":  true,
		"m5ad": true,
		"m5d":  true,
		"m5dn": true,
		"m5n":  true,
		"m5zn": true,
		"m6g":  true,
		"m6gd": true,
		"p3dn": true,
		"p4":   true,
		"r5":   true,
		"r5a":  true,
		"r5ad": true,
		"r5b":  true,
		"r5d":  true,
		"r5dn": true,
		"r5n":  true,
		"r6g":  true,
		"r6gd": true,
		"t3":   true,
		"t3a":  true,
		"t4g":  true,
		"z1d":  true,
	}
)

// GetEnaSupportForFlavor checks whether an image should be registered with EnaSupport based on instances type which will load the image
func GetEnaSupportForFlavor(flavor string) bool {
	if flavor == "" {
		return false
	}

	flavorParts := strings.Split(flavor, ".")

	instanceFamily := strings.ToLower(flavorParts[0])

	_, exists := NitroInstanceTypes[instanceFamily]
	return exists
}

func getAWSImages(ec2Service *ec2.EC2) (*ec2.DescribeImagesOutput, error) {
	filters := []*ec2.Filter{{Name: aws.String("tag:CreatedBy"), Values: aws.StringSlice([]string{"ops"})}}

	input := &ec2.DescribeImagesInput{
		Owners: []*string{
			aws.String("self"),
		},
		Filters: filters,
	}

	result, err := ec2Service.DescribeImages(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return nil, errors.New(aerr.Error())
			}
		} else {
			return nil, errors.New(err.Error())
		}
	}

	return result, nil
}

// GetImages return all images on AWS
func (p *AWS) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	var cimages []lepton.CloudImage

	result, err := getAWSImages(p.ec2)
	if err != nil {
		return nil, err
	}

	images := result.Images
	for _, image := range images {
		tagName := p.getNameTag(image.Tags)

		var name string

		if tagName != nil {
			name = *tagName.Value
		} else {
			name = "n/a"
		}

		imageCreatedAt, _ := time.Parse("2006-01-02T15:04:05Z", *image.CreationDate)

		cimage := lepton.CloudImage{
			Name:    name,
			ID:      *image.Name,
			Status:  *image.State,
			Created: imageCreatedAt,
		}

		cimages = append(cimages, cimage)
	}

	return cimages, nil
}

// ListImages lists images on AWS
func (p *AWS) ListImages(ctx *lepton.Context) error {
	cimages, err := p.GetImages(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range cimages {
		var row []string

		row = append(row, image.Name)
		row = append(row, image.ID)
		row = append(row, image.Status)
		row = append(row, lepton.Time2Human(image.Created))

		table.Append(row)
	}

	table.Render()

	return nil
}

// ResizeImage is not supported on AWS.
func (p *AWS) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// DeleteImage deletes image from AWS by ami name
func (p *AWS) DeleteImage(ctx *lepton.Context, imagename string) error {
	// delete ami by ami name
	image, err := p.findImageByName(imagename)
	if err != nil {
		return fmt.Errorf("Error running deregister image operation: %s", err)
	}

	amiID := aws.StringValue(image.ImageId)
	snapID := aws.StringValue(image.BlockDeviceMappings[0].Ebs.SnapshotId)

	// grab snapshotid && grab image id

	params := &ec2.DeregisterImageInput{
		ImageId: aws.String(amiID),
		DryRun:  aws.Bool(false),
	}
	_, err = p.ec2.DeregisterImage(params)
	if err != nil {
		return fmt.Errorf("Error running deregister image operation: %s", err)
	}

	// DeleteSnapshot
	params2 := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapID),
		DryRun:     aws.Bool(false),
	}
	_, err = p.ec2.DeleteSnapshot(params2)
	if err != nil {
		return fmt.Errorf("Error running snapshot delete: %s", err)
	}

	return nil
}

func (p *AWS) findImageByName(name string) (*ec2.Image, error) {
	ec2Filters := []*ec2.Filter{
		{Name: aws.String("tag:Name"), Values: []*string{&name}},
		{Name: aws.String("tag:CreatedBy"), Values: []*string{aws.String("ops")}},
	}

	input := &ec2.DescribeImagesInput{
		Filters: ec2Filters,
	}

	result, err := p.ec2.DescribeImages(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Error(aerr)
			}
		} else {
			log.Error(err)
		}
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("image %v not found", name)
	}

	return result.Images[0], nil
}

// SyncImage syncs image from provider to another provider
func (p *AWS) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (p *AWS) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.Imagename
	return imagePath, nil
}

// not an archive - just raw disk image
func (p *AWS) getArchiveName(ctx *lepton.Context) string {
	imagePath := ctx.Config().RunConfig.Imagename
	return imagePath
}

func (p *AWS) waitSnapshotToBeReady(config *types.Config, importTaskID *string) (*string, error) {
	taskFilter := &ec2.DescribeImportSnapshotTasksInput{
		ImportTaskIds: []*string{importTaskID},
	}

	_, err := p.ec2.DescribeImportSnapshotTasks(taskFilter)
	if err != nil {
		return nil, err
	}

	log.Info("waiting for snapshot - can take like 5 min...")

	waitStartTime := time.Now()
	bar := progressbar.New(100)
	bar.RenderBlank()

	ct := aws.BackgroundContext()
	w := request.Waiter{
		Name:        "DescribeImportSnapshotTasks",
		Delay:       request.ConstantWaiterDelay(15 * time.Second),
		MaxAttempts: 120,
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
			// update progress bar
			snapshotTasksOutput, err := p.ec2.DescribeImportSnapshotTasks(taskFilter)
			if err == nil && len(snapshotTasksOutput.ImportSnapshotTasks) > 0 {
				snapshotProgress := (*snapshotTasksOutput.ImportSnapshotTasks[0]).SnapshotTaskDetail.Progress

				if snapshotProgress != nil {
					progress, _ := strconv.Atoi(*snapshotProgress)
					bar.Set(progress)
					bar.RenderBlank()
				}
			}

			req, _ := p.ec2.DescribeImportSnapshotTasksRequest(taskFilter)
			req.SetContext(ct)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}

	err = w.WaitWithContext(ct)

	bar.Set(100)
	bar.Finish()
	bar.RenderBlank()

	if err != nil {
		fmt.Printf("\nimport timed out after %f minutes\n", time.Since(waitStartTime).Minutes())
		return nil, err
	}

	fmt.Printf("\nimport done - took %f minutes\n", time.Since(waitStartTime).Minutes())

	describeOutput, err := p.ec2.DescribeImportSnapshotTasks(taskFilter)
	if err != nil {
		return nil, err
	}

	snapshotID := describeOutput.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId

	return snapshotID, nil
}
