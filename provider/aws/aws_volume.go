//go:build aws || !onlyprovider

package aws

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nanovms/ops/lepton"
)

// CreateVolume creates a snapshot and use it to create a volume
func (a *AWS) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	config := ctx.Config()
	var sizeInGb int64
	var vol lepton.NanosVolume
	if config.BaseVolumeSz != "" {
		size, err := lepton.GetSizeInGb(config.BaseVolumeSz)
		if err != nil {
			return vol, fmt.Errorf("cannot get volume size: %v", err)
		}
		config.BaseVolumeSz = "" // create minimum-sized local volume
		sizeInGb = int64(size)
	}

	// Create volume
	vol, err := lepton.CreateLocalVolume(config, name, data, provider)
	if err != nil {
		return vol, fmt.Errorf("create local volume: %v", err)
	}
	defer os.Remove(vol.Path)

	config.CloudConfig.ImageName = vol.Name

	err = a.Storage.CopyToBucket(config, vol.Path)
	if err != nil {
		return vol, fmt.Errorf("copy volume archive to aws bucket: %v", err)
	}

	bucket := config.CloudConfig.BucketName
	key := vol.Name

	input := &ec2.ImportSnapshotInput{
		Description: aws.String("name"),
		DiskContainer: &ec2.SnapshotDiskContainer{
			Description: aws.String("snapshot imported"),
			Format:      aws.String("raw"),
			UserBucket: &ec2.UserBucket{
				S3Bucket: aws.String(bucket),
				S3Key:    aws.String(key),
			},
		},
	}

	res, err := a.ec2.ImportSnapshot(input)
	if err != nil {
		return vol, fmt.Errorf("import snapshot: %v", err)
	}

	snapshotID, err := a.waitSnapshotToBeReady(config, res.ImportTaskId)
	if err != nil {
		return vol, err
	}

	// delete the tmp s3 volume
	err = a.Storage.DeleteFromBucket(config, key)
	if err != nil {
		return vol, err
	}

	// Create tags to assign to the volume
	tags, _ := buildAwsTags(config.CloudConfig.Tags, name)

	// Create volume from snapshot
	createVolumeInput := &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(config.CloudConfig.Zone),
		SnapshotId:       snapshotID,
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("volume"),
				Tags:         tags,
			},
		},
	}
	if sizeInGb != 0 {
		createVolumeInput.Size = &sizeInGb
	}
	_, err = a.ec2.CreateVolume(createVolumeInput)
	if err != nil {
		return vol, fmt.Errorf("create aws volume: %v", err)
	}

	return vol, nil
}

// GetAllVolumes finds and returns all volumes
func (a *AWS) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	vols := &[]lepton.NanosVolume{}

	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:CreatedBy"), Values: []*string{aws.String("ops")}},
		},
	}

	output, err := a.ec2.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}

	for _, volume := range output.Volumes {
		var name string
		var attachments []string

		for _, tag := range volume.Tags {
			if *tag.Key == "Name" {
				name = *tag.Value
			}
		}

		for _, att := range volume.Attachments {
			attachments = append(attachments, *att.InstanceId)
		}

		vol := lepton.NanosVolume{
			ID:         *volume.VolumeId,
			Name:       name,
			Status:     *volume.State,
			Size:       strconv.Itoa(int(*volume.Size)),
			Path:       "",
			CreatedAt:  volume.CreateTime.String(),
			AttachedTo: strings.Join(attachments, ";"),
		}

		*vols = append(*vols, vol)
	}

	return vols, nil
}

// DeleteVolume deletes a volume
func (a *AWS) DeleteVolume(ctx *lepton.Context, name string) error {
	vol, err := a.findVolumeByName(name)
	if err != nil {
		return err
	}

	input := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(*vol.VolumeId),
	}
	_, err = a.ec2.DeleteVolume(input)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches a volume to an instance
func (a *AWS) AttachVolume(ctx *lepton.Context, instanceName, name string, attachID int) error {
	vol, err := a.findVolumeByName(name)
	if err != nil {
		return err
	}

	instance, err := a.findInstanceByName(instanceName)
	if err != nil {
		return err
	}

	device := ""
	if attachID >= 0 {
		if attachID >= 1 && attachID <= 25 {
			device = "/dev/sd" + string(rune('a'+attachID))
		} else {
			return fmt.Errorf("invalid attachment point %d; allowed values: 1-25", attachID)
		}
	} else {
		// Look for an unused device name to be assigned to the volume, starting from "/dev/sdb"
		for deviceLetter := 'b'; deviceLetter <= 'z'; deviceLetter++ {
			name := "/dev/sd" + string(deviceLetter)
			nameUsed := false
			for _, mapping := range instance.BlockDeviceMappings {
				if *mapping.DeviceName == name {
					nameUsed = true
					break
				}
			}
			if !nameUsed {
				device = name
				break
			}
		}
		if device == "" {
			return errors.New("no available device names")
		}
	}

	input := &ec2.AttachVolumeInput{
		Device:     aws.String(device),
		InstanceId: aws.String(*instance.InstanceId),
		VolumeId:   aws.String(*vol.VolumeId),
	}
	_, err = a.ec2.AttachVolume(input)
	if err != nil {
		return err
	}

	return nil
}

// DetachVolume detachs a volume from an instance
func (a *AWS) DetachVolume(ctx *lepton.Context, instanceName, name string) error {
	vol, err := a.findVolumeByName(name)
	if err != nil {
		return err
	}

	instance, err := a.findInstanceByName(instanceName)
	if err != nil {
		return err
	}

	input := &ec2.DetachVolumeInput{
		InstanceId: aws.String(*instance.InstanceId),
		VolumeId:   aws.String(*vol.VolumeId),
	}

	_, err = a.ec2.DetachVolume(input)
	if err != nil {
		return err
	}

	return nil
}

func (a *AWS) findVolumeByName(name string) (*ec2.Volume, error) {
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:CreatedBy"), Values: []*string{aws.String("ops")}},
		},
	}

	output, err := a.ec2.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}

	for _, volume := range output.Volumes {
		if *volume.VolumeId == name {
			return volume, nil
		}
		for _, tag := range volume.Tags {
			if (*tag.Key == "Name") && (*tag.Value == name) {
				return volume, nil
			}
		}
	}

	return nil, fmt.Errorf("volume '%s' not found", name)
}

// CreateVolumeImage ...
func (a *AWS) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (a *AWS) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
