package lepton

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
)

func (o *OpenStack) getVolumesClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}

// CreateVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) CreateVolume(ctx *Context, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume

	imagesClient, err := o.getImagesClient()
	if err != nil {
		return vol, err
	}

	image, err := o.createImage(imagesClient, name)
	if err != nil {
		return vol, err
	}

	vol, err = CreateLocalVolume(ctx.config, name, data, size, provider)
	if err != nil {
		return vol, err
	}

	err = o.uploadImage(imagesClient, image.ID, vol.Path)
	if err != nil {
		return vol, err
	}

	volumesClient, err := o.getVolumesClient()
	if err != nil {
		return vol, err
	}

	sizeNum, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return vol, err
	}

	sizeNum = sizeNum / 1000 / 1000 / 1000 // convert B to GB
	if sizeNum < 1 {
		sizeNum = 1
	}

	createOpts := volumes.CreateOpts{
		Name:    name,
		Size:    int(math.Round(sizeNum)),
		ImageID: image.ID,
	}

	response := volumes.Create(volumesClient, createOpts)

	_, err = response.Extract()
	if err != nil {
		return vol, err
	}

	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (o *OpenStack) GetAllVolumes(ctx *Context) (*[]NanosVolume, error) {
	var vols []NanosVolume

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
		var name string
		var attachments []string

		for _, att := range volume.Attachments {
			attachments = append(attachments, att.HostName)
		}

		vol := NanosVolume{
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
func (o *OpenStack) DeleteVolume(ctx *Context, name string) error {
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

	imageID, err := o.findImage(name)
	if err != nil {
		return err
	}

	imagesClient, err := o.getImagesClient()
	if err != nil {
		return err
	}

	err = o.deleteImage(imagesClient, imageID)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) AttachVolume(ctx *Context, image, name, mount string) error {
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
		Device:   "/dev/" + mount,
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
		exitWithError("Failed to soft reboot instance.")
	}

	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (o *OpenStack) DetachVolume(ctx *Context, image, name string) error {
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
