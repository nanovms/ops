package aws

import (
	"bytes"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"

	"github.com/schollz/progressbar/v3"
)

const (
	// SnapshotBlockDataLength define block length 512K
	SnapshotBlockDataLength = 524288

	// PutSnapshotBlockLimit define limit requests per snapshot, each supported Region: 1,000 per second
	// https://docs.aws.amazon.com/general/latest/gr/ebs-service.html#limits_ebs
	// We config 80% of max for smooth concurrent
	PutSnapshotBlockLimit = 800
)

// PutSnapshotBlockResult define result from PutSnapshotBlock
type PutSnapshotBlockResult struct {
	Error      error
	BlockIndex int64
}

// PutSnapshotBlockResults define array of PutSnapshotBlockResult
type PutSnapshotBlockResults struct {
	Data []PutSnapshotBlockResult

	// sync.Mutex to lock the slice
	sync.Mutex
}

// Set a PutSnapshotBlockResult to slice Data
func (psb *PutSnapshotBlockResults) Set(result PutSnapshotBlockResult) {
	psb.Lock()
	defer psb.Unlock()

	psb.Data = append(psb.Data, result)
}

// Len get len of data
func (psb *PutSnapshotBlockResults) Len() int {
	psb.Lock()
	defer psb.Unlock()

	return len(psb.Data)
}

// BuildImage to be upload on AWS
func (p *AWS) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()

	if isGravitonProcessor(c.CloudConfig.Flavor) {
		c.Uefi = true
	}

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

// CreateImage creates image on AWS using nanos images.
// First a snapshot in AWS is created from a image in local machine and, then the snapshot is used to create an AMI.
func (p *AWS) CreateImage(ctx *lepton.Context, imagePath string) error {
	imageName := ctx.Config().CloudConfig.ImageName

	i, _ := p.findImageByName(imageName)
	if i != nil {
		return fmt.Errorf("failed creating image: image with name %s already exists", imageName)
	}

	c := ctx.Config()

	key := c.CloudConfig.ImageName

	ctx.Logger().Info("Creating snapshot")
	snapshotID, err := p.createSnapshot(imagePath)
	if err != nil {
		return err
	}

	// tag the volume
	tags, _ := buildAwsTags(c.CloudConfig.Tags, key)

	ctx.Logger().Info("Tagging snapshot")
	_, err = p.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: aws.StringSlice([]string{snapshotID}),
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
		Architecture: aws.String(getArchitecture(c.CloudConfig.Flavor)),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          aws.String(snapshotID),
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
	if err != nil {
		return err
	}

	return nil
}

// MirrorImage copies an image using its imageName from one region to another
func (p *AWS) MirrorImage(ctx *lepton.Context, imageName, srcRegion, dstRegion string) (string, error) {
	srcSession, err := session.NewSession(
		&aws.Config{
			Region: aws.String(stripZone(srcRegion)),
		},
	)
	if err != nil {
		return "", err
	}

	srcEc2 := ec2.New(srcSession)

	i, err := p.findImageByNameUsingSession(srcEc2, imageName)
	if i == nil {
		return "", fmt.Errorf("no image with name %s found", imageName)
	}
	if err != nil {
		return "", fmt.Errorf("error while search for image: %s", err.Error())
	}

	dstSession, err := session.NewSession(
		&aws.Config{
			Region: aws.String(stripZone(dstRegion)),
		},
	)
	if err != nil {
		return "", err
	}

	dstEc2 := ec2.New(dstSession)

	output, err := dstEc2.CopyImage(&ec2.CopyImageInput{
		Name:          aws.String(imageName),
		SourceImageId: i.ImageId,
		SourceRegion:  &srcRegion,
	})
	if err != nil {
		return "", err
	}

	tags, _ := buildAwsTags(ctx.Config().CloudConfig.Tags, imageName)

	dstEc2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{output.ImageId},
		Tags:      tags,
	})

	if err != nil {
		return "", err
	}
	return *output.ImageId, nil
}

// createSnapshot process create Snapshot to EBS
// Returns snapshotID and err
func (p *AWS) createSnapshot(imagePath string) (string, error) {
	// Open file first
	f, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// create progressBar track put snapshot
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	// maxBar include process of createSnapshot, completeSnapshot, putSnapshot (include request and response from ebs api)
	maxBar := (fi.Size()/int64(SnapshotBlockDataLength))*2 + 2
	bar := progressbar.Default(maxBar)

	snapshotOutput, err := p.volumeService.StartSnapshot(&ebs.StartSnapshotInput{
		Tags:       []*ebs.Tag{},
		VolumeSize: aws.Int64(1),
	})
	if err != nil {
		return "", err
	}

	bar.Add64(1)

	snapshotID := *snapshotOutput.SnapshotId

	blockIndex := int64(0)
	var snapshotBlocksChecksums []byte
	wg := sync.WaitGroup{}
	chanBlockResult := make(chan PutSnapshotBlockResult)
	var blockResults PutSnapshotBlockResults
	done := make(chan bool)

	go func() {
		for result := range chanBlockResult {
			if result.Error == nil {
				// when success add one to bar
				bar.Add64(1)
			}
			blockResults.Set(result)
		}
		done <- true
	}()

	for {
		block := make([]byte, SnapshotBlockDataLength)

		if _, err := f.Read(block); err == io.EOF {
			break
		} else if err != nil {
			return snapshotID, err
		}

		input, checksum := buildSnapshotBlockInput(*snapshotOutput.SnapshotId, blockIndex, block)
		snapshotBlocksChecksums = append(snapshotBlocksChecksums, checksum...)

		wg.Add(1)
		go p.writeToBlock(input, &wg, chanBlockResult)

		blockIndex++

		// when PutSnapshotBlock add one to bar
		bar.Add64(1)

		if blockIndex%PutSnapshotBlockLimit == 0 {
			// When concurrent reach PutSnapshotBlockLimit, we waiting for finish
			for {
				if blockIndex == int64(blockResults.Len()) {
					break
				}

				time.Sleep(2 * time.Second)
			}
		}
	}

	wg.Wait()
	close(chanBlockResult)

	<-done
	close(done)

	if err := p.retryPutSnapshotBlocks(bar, f, snapshotID, &blockResults); err != nil {
		return snapshotID, err
	}

	h := sha256.New()
	h.Write(snapshotBlocksChecksums)
	snapshotChecksum := b64.StdEncoding.EncodeToString(h.Sum(nil))

	if _, err := p.volumeService.CompleteSnapshot(&ebs.CompleteSnapshotInput{
		ChangedBlocksCount:        &blockIndex,
		Checksum:                  aws.String(snapshotChecksum),
		ChecksumAggregationMethod: aws.String("LINEAR"),
		ChecksumAlgorithm:         aws.String("SHA256"),
		SnapshotId:                aws.String(snapshotID),
	}); err != nil {
		return snapshotID, err
	}

	bar.Add64(1)

	if err := p.ec2.WaitUntilSnapshotCompleted(&ec2.DescribeSnapshotsInput{
		SnapshotIds: aws.StringSlice([]string{*snapshotOutput.SnapshotId}),
	}); err != nil {
		return snapshotID, err
	}

	return snapshotID, nil
}

// retryPutSnapshotBlocks if any error from BlockResults, we get data from the file again and try PutSnapshotBlock sequentially
// Returns an error
func (p *AWS) retryPutSnapshotBlocks(bar *progressbar.ProgressBar, f *os.File, snapshotID string, blockResults *PutSnapshotBlockResults) error {
	var errs []error
	for _, data := range blockResults.Data {
		if data.Error != nil {
			block := make([]byte, SnapshotBlockDataLength)

			if _, err := f.ReadAt(block, data.BlockIndex*SnapshotBlockDataLength); err != nil {
				return err
			}
			input, _ := buildSnapshotBlockInput(snapshotID, data.BlockIndex, block)

			log.Debug("RetryPutSnapshotBlock", data.BlockIndex, "PreviousErr", data.Error)
			if _, err := p.volumeService.PutSnapshotBlock(input); err != nil {
				errs = append(errs, err)
			}

			bar.Add64(1)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (p *AWS) writeToBlock(input *ebs.PutSnapshotBlockInput, wg *sync.WaitGroup, chanBlockResult chan PutSnapshotBlockResult) {
	defer wg.Done()

	_, err := p.volumeService.PutSnapshotBlock(input)
	chanBlockResult <- PutSnapshotBlockResult{
		Error:      err,
		BlockIndex: *input.BlockIndex,
	}
}

// buildSnapshotBlockInput
func buildSnapshotBlockInput(snapshotID string, blockIndex int64, block []byte) (*ebs.PutSnapshotBlockInput, []byte) {
	h := sha256.New()
	h.Write(block)
	checksum := b64.StdEncoding.EncodeToString(h.Sum(nil))

	return &ebs.PutSnapshotBlockInput{
		BlockData:         aws.ReadSeekCloser(bytes.NewReader(block)),
		BlockIndex:        aws.Int64(blockIndex),
		Checksum:          aws.String(checksum),
		ChecksumAlgorithm: aws.String("SHA256"),
		DataLength:        aws.Int64(SnapshotBlockDataLength),
		SnapshotId:        aws.String(snapshotID),
	}, h.Sum(nil)
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
		"c6a":  true,
		"c6g":  true,
		"c6gd": true,
		"c6gn": true,
		"d3":   true,
		"d3en": true,
		"g4dn": true,
		"g5":   true,
		"i3en": true,
		"inf1": true,
		"m5":   true,
		"m5a":  true,
		"m5ad": true,
		"m5d":  true,
		"m5dn": true,
		"m5n":  true,
		"m5zn": true,
		"m6a":  true,
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

	// Architectures define architecture from instance family
	Architectures = map[string]string{
		"a1":      "arm64",
		"c1":      "x86_64",
		"c3":      "x86_64",
		"c4":      "x86_64",
		"c5":      "x86_64",
		"c5a":     "x86_64",
		"c5ad":    "x86_64",
		"c5d":     "x86_64",
		"c5n":     "x86_64",
		"c6a":     "x86_64",
		"c6g":     "arm64",
		"c6gd":    "arm64",
		"c6gn":    "arm64",
		"c6i":     "x86_64",
		"c7g":     "arm64",
		"cc2":     "x86_64",
		"d3":      "x86_64",
		"d3en":    "x86_64",
		"dl1":     "x86_64",
		"f1":      "x86_64",
		"g2":      "x86_64",
		"g3":      "x86_64",
		"g3s":     "x86_64",
		"g4ad":    "x86_64",
		"g4dn":    "x86_64",
		"g5":      "x86_64",
		"g5g":     "arm64",
		"h1":      "x86_64",
		"i2":      "x86_64",
		"i3":      "x86_64",
		"i3en":    "x86_64",
		"im4gn":   "arm64",
		"inf1":    "x86_64",
		"is4gen":  "arm64",
		"m1":      "x86_64",
		"m2":      "x86_64",
		"m3":      "x86_64",
		"m4":      "x86_64",
		"m5":      "x86_64",
		"m5a":     "x86_64",
		"m5ad":    "x86_64",
		"m5d":     "x86_64",
		"m5dn":    "x86_64",
		"m5n":     "x86_64",
		"m5zn":    "x86_64",
		"m6a":     "x86_64",
		"m6g":     "arm64",
		"m6gd":    "arm64",
		"m6i":     "x86_64",
		"mac1":    "x86_64_mac",
		"p2":      "x86_64",
		"p3":      "x86_64",
		"p3dn":    "x86_64",
		"p4d":     "x86_64",
		"r3":      "x86_64",
		"r4":      "x86_64",
		"r5":      "x86_64",
		"r5a":     "x86_64",
		"r5ad":    "x86_64",
		"r5b":     "x86_64",
		"r5d":     "x86_64",
		"r5dn":    "x86_64",
		"r5n":     "x86_64",
		"r6g":     "arm64",
		"r6gd":    "arm64",
		"r6i":     "x86_64",
		"t1":      "i386,",
		"t2":      "x86_64",
		"t3":      "x86_64",
		"t3a":     "x86_64",
		"t4g":     "arm64",
		"u-12tb1": "x86_64",
		"u-6tb1":  "x86_64",
		"u-9tb1":  "x86_64",
		"vt1":     "x86_64",
		"x1":      "x86_64",
		"x1e":     "x86_64",
		"x2gd":    "arm64",
		"x2iezn":  "x86_64",
		"z1d":     "x86_64",
	}

	// GravitonProcessors are designed by AWS to deliver the best price performance for your cloud workloads running in Amazon EC2
	GravitonProcessors = map[string]bool{"a1": true, "c6g": true, "c6gd": true, "c6gn": true, "c7g": true, "g5g": true, "im4gn": true, "is4gen": true, "m6g": true, "m6gd": true, "r6g": true, "r6gd": true, "t4g": true, "x2gd": true}
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

func getArchitecture(flavor string) string {
	if flavor == "" {
		return "x86_64"
	}

	if arc, isExist := Architectures[strings.ToLower(strings.Split(flavor, ".")[0])]; isExist {
		return arc
	}
	return "x86_64"
}

func isGravitonProcessor(flavor string) bool {
	if flavor == "" {
		return false
	}

	_, isExist := GravitonProcessors[strings.ToLower(strings.Split(flavor, ".")[0])]
	return isExist
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

		if tagName == nil {
			tagName = &ec2.Tag{Value: aws.String("n/a")}
		}

		imageCreatedAt, _ := time.Parse("2006-01-02T15:04:05Z", *image.CreationDate)

		cimage := lepton.CloudImage{
			Tag:     *tagName.Value,
			Name:    *image.Name,
			ID:      *image.ImageId,
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

	if ctx.Config().RunConfig.JSON {
		// json output
		if err := json.NewEncoder(os.Stdout).Encode(cimages); err != nil {
			return err
		}
	} else {
		// default of table output
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"AmiID", "AmiName", "Name", "Status", "Created"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
		table.SetRowLine(true)

		for _, image := range cimages {
			var row []string

			row = append(row, image.ID)
			row = append(row, image.Name)
			row = append(row, image.Tag)
			row = append(row, image.Status)
			row = append(row, lepton.Time2Human(image.Created))

			table.Append(row)
		}

		table.Render()
	}
	return nil
}

// ResizeImage is not supported on AWS.
func (p *AWS) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// DeleteImage deletes image from AWS by ami name
func (p *AWS) DeleteImage(ctx *lepton.Context, imagename string) error {
	// delete ami by ami name
	image, err := p.findImageByName(imagename)
	if err != nil {
		return fmt.Errorf("error running deregister image operation: %s", err)
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
		return fmt.Errorf("error running deregister image operation: %s", err)
	}

	// DeleteSnapshot
	params2 := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapID),
		DryRun:     aws.Bool(false),
	}
	_, err = p.ec2.DeleteSnapshot(params2)
	if err != nil {
		return fmt.Errorf("error running snapshot delete: %s", err)
	}

	return nil
}

func (p *AWS) findImageByName(name string) (*ec2.Image, error) {
	return p.findImageByNameUsingSession(p.ec2, name)
}

func (p *AWS) findImageByNameUsingSession(ec2Session *ec2.EC2, name string) (*ec2.Image, error) {
	ec2Filters := []*ec2.Filter{
		{Name: aws.String("tag:Name"), Values: []*string{&name}},
		{Name: aws.String("tag:CreatedBy"), Values: []*string{aws.String("ops")}},
	}

	input := &ec2.DescribeImagesInput{
		Filters: ec2Filters,
	}

	result, err := ec2Session.DescribeImages(input)
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
