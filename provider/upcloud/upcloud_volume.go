//go:build upcloud || !onlyprovider

package upcloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/request"
	"github.com/nanovms/ops/lepton"
)

// CreateVolume creates a local volume and uploads the volume to upcloud
func (p *Provider) CreateVolume(ctx *lepton.Context, name, data, provider string) (vol lepton.NanosVolume, err error) {
	vol, err = lepton.CreateLocalVolume(ctx.Config(), name, data, provider)
	if err != nil {
		return
	}
	defer os.Remove(vol.Path)

	storageDetails, err := p.createStorage(ctx, name, vol.Path)
	if err != nil {
		return
	}

	ctx.Logger().Debugf("%+v", storageDetails)

	return vol, nil
}

// GetAllVolumes returns every upcloud volume
func (p *Provider) GetAllVolumes(ctx *lepton.Context) (volumes *[]lepton.NanosVolume, err error) {
	volumes = &[]lepton.NanosVolume{}

	listTemplatesReq := &request.GetStoragesRequest{
		Type:   "disk",
		Access: "private",
	}

	templates, err := p.upcloud.GetStorages(context.Background(), listTemplatesReq)
	if err != nil {
		return
	}

	ctx.Logger().Debugf("%+v", templates)

	volumesCh := make(chan *upcloud.StorageDetails)
	errCh := make(chan error)

	for _, s := range templates.Storages {
		go p.asyncGetVolume(s.UUID, volumesCh, errCh)
	}

	for i := 0; i < len(templates.Storages); i++ {
		select {
		case s := <-volumesCh:
			*volumes = append(*volumes, lepton.NanosVolume{
				ID:         s.UUID,
				Name:       s.Title,
				Status:     s.State,
				Size:       strconv.Itoa(s.Size),
				AttachedTo: strings.Join(s.ServerUUIDs, ","),
				CreatedAt:  lepton.Time2Human(s.Created),
			})
		case err = <-errCh:
			return
		}
	}

	return
}

func (p *Provider) asyncGetVolume(uuid string, volumesCh chan *upcloud.StorageDetails, errCh chan error) {
	volumeReq := &request.GetStorageDetailsRequest{
		UUID: uuid,
	}

	details, err := p.upcloud.GetStorageDetails(context.Background(), volumeReq)
	if err != nil {
		errCh <- err
		return
	}

	volumesCh <- details
}

// DeleteVolume deletes a volume from upcloud
func (p *Provider) DeleteVolume(ctx *lepton.Context, name string) (err error) {
	volume, err := p.getVolumeByName(ctx, name)
	if err != nil {
		return
	}

	err = p.deleteStorage(volume.ID)

	return
}

// AttachVolume attaches a storage to an upcloud server
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name string, attachID int) (err error) {
	instance, err := p.GetInstanceByName(ctx, image)
	if err != nil {
		return
	}

	if instance.Status != "stopped" {
		ctx.Logger().Log("stopping instance")
		err = p.stopServer(instance.ID)
		if err != nil {
			return
		}

		err = p.waitForServerState(instance.ID, "stopped")
		if err != nil {
			return
		}
	}

	volume, err := p.getVolumeByName(ctx, name)
	if err != nil {
		return
	}

	ctx.Logger().Log("attaching volume")
	attachReq := &request.AttachStorageRequest{
		ServerUUID:  instance.ID,
		StorageUUID: volume.ID,
	}

	_, err = p.upcloud.AttachStorage(context.Background(), attachReq)

	ctx.Logger().Log("starting instance")
	err = p.startServer(instance.ID)

	return
}

// DetachVolume detaches a storage from an upcloud server
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) (err error) {
	server, err := p.getServerByName(ctx, image)
	if err != nil {
		return
	}

	serverDetailsReq := &request.GetServerDetailsRequest{UUID: server.UUID}

	serverDetails, err := p.upcloud.GetServerDetails(context.Background(), serverDetailsReq)
	if err != nil {
		return
	}

	ctx.Logger().Log("detaching volume")

	for _, s := range serverDetails.StorageDevices {
		if s.Title == name {
			if server.State != "stopped" {
				ctx.Logger().Log("stopping server")
				err = p.stopServer(server.UUID)
				if err != nil {
					return
				}

				err = p.waitForServerState(server.UUID, "stopped")
				if err != nil {
					return
				}
			}

			detachReq := &request.DetachStorageRequest{
				ServerUUID: server.UUID,
				Address:    s.Address,
			}

			_, err = p.upcloud.DetachStorage(context.Background(), detachReq)

			ctx.Logger().Log("starting server")
			err = p.startServer(server.UUID)
			return
		}
	}

	err = fmt.Errorf(`volume "%s" is not attached to server "%s"`, name, image)

	return
}

func (p *Provider) getVolumeByName(ctx *lepton.Context, volumeName string) (volume *lepton.NanosVolume, err error) {
	vols, err := p.GetAllVolumes(ctx)
	if err != nil {
		return
	}

	for _, v := range *vols {
		if v.Name == volumeName {
			volume = &v
			return
		}
	}

	err = errors.New("volume not found")

	return
}

// CreateVolumeImage ...
func (p *Provider) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (p *Provider) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
