//go:build openstack || !onlyprovider

package openstack

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

func (o *OpenStack) getVolumesClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}

// CreateVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	var vol lepton.NanosVolume

	imagesClient, err := o.getImagesClient()
	if err != nil {
		return vol, err
	}

	image, err := o.createImage(imagesClient, name)
	if err != nil {
		return vol, err
	}

	vol, err = lepton.CreateLocalVolume(ctx.Config(), name, data, provider)
	if err != nil {
		return vol, err
	}
	defer os.Remove(vol.Path)

	err = o.uploadImage(imagesClient, image.ID, vol.Path)
	if err != nil {
		return vol, err
	}

	volumesClient, err := o.getVolumesClient()
	if err != nil {
		return vol, err
	}

	sizeInGb, err := lepton.GetSizeInGb(ctx.Config().BaseVolumeSz)
	if err != nil {
		return vol, err
	}
	if sizeInGb < 1 {
		sizeInGb = 1
	}

	createOpts := volumes.CreateOpts{
		Name:    name,
		Size:    sizeInGb,
		ImageID: image.ID,
	}

	response := volumes.Create(volumesClient, createOpts)

	r, err := response.Extract()
	if err != nil {
		return vol, err
	}

	log.Info("creating volume...")
	err = volumes.WaitForStatus(volumesClient, r.ID, "available", 60)
	if err != nil {
		return vol, err
	}

	err = o.deleteImage(imagesClient, image.ID)
	if err != nil {
		return vol, err
	}

	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (o *OpenStack) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	var vols []lepton.NanosVolume

	client, err := o.getVolumesClient()
	if err != nil {
		return nil, err
	}

	allPages, err := volumes.List(client, volumes.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}

	volumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return nil, err
	}

	for _, volume := range volumes {
		name := volume.Name
		var attachments []string

		for _, att := range volume.Attachments {
			attachments = append(attachments, att.HostName)
		}

		vol := lepton.NanosVolume{
			ID:         volume.ID,
			Name:       name,
			Status:     volume.Status,
			CreatedAt:  volume.CreatedAt.String(),
			AttachedTo: strings.Join(attachments, ";"),
			Size:       strconv.Itoa(volume.Size),
		}

		vols = append(vols, vol)
	}

	return &vols, nil
}

func (o *OpenStack) getVolumeByName(volumesClient *gophercloud.ServiceClient, name string) (*volumes.Volume, error) {
	allPages, err := volumes.List(volumesClient, volumes.ListOpts{Name: name}).AllPages()
	if err != nil {
		return nil, err
	}

	volumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return nil, err
	}

	if len(volumes) == 0 {
		return nil, errors.New("Volume not found")
	}

	return &volumes[0], nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) DeleteVolume(ctx *lepton.Context, name string) error {
	volumesClient, err := o.getVolumesClient()
	if err != nil {
		return err
	}

	volume, err := o.getVolumeByName(volumesClient, name)
	if err != nil {
		return err
	}

	opts := volumes.DeleteOpts{
		Cascade: true,
	}

	err = volumes.Delete(volumesClient, volume.ID, opts).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	computeClient, err := o.getComputeClient()
	if err != nil {
		return err
	}

	volumesClient, err := o.getVolumesClient()
	if err != nil {
		return err
	}

	volume, err := o.getVolumeByName(volumesClient, name)
	if err != nil {
		return err
	}

	server, err := o.findInstance(image)
	if err != nil {
		return err
	}

	createOpts := volumeattach.CreateOpts{
		Device:   "/dev/" + name,
		VolumeID: volume.ID,
	}

	_, err = volumeattach.Create(computeClient, server.ID, createOpts).Extract()

	if err != nil {
		return err
	}

	rebootOpts := servers.RebootOpts{
		Type: servers.SoftReboot,
	}

	errReboot := servers.Reboot(computeClient, server.ID, rebootOpts).ExtractErr()

	if errReboot != nil {
		return errors.New("failed to soft reboot instance")
	}

	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) DetachVolume(ctx *lepton.Context, image, name string) error {
	computeClient, err := o.getComputeClient()
	if err != nil {
		return err
	}

	volumesClient, err := o.getVolumesClient()
	if err != nil {
		return err
	}

	volume, err := o.getVolumeByName(volumesClient, name)
	if err != nil {
		return err
	}

	server, err := o.findInstance(image)
	if err != nil {
		return err
	}

	for _, attachment := range volume.Attachments {
		if attachment.ServerID == server.ID {
			err := volumeattach.Delete(computeClient, server.ID, attachment.ID).ExtractErr()
			return err
		}
	}

	return fmt.Errorf("volume %v is not attached to instance %v", name, image)
}

// CreateVolumeImage ...
func (o *OpenStack) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (o *OpenStack) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
