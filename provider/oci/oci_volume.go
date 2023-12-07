//go:build oci || !onlyprovider

package oci

import (
	"context"
	"errors"
	"strconv"

	"github.com/nanovms/ops/lepton"
	"github.com/oracle/oci-go-sdk/core"
)

// CreateVolume creates a local volume and uploads the volume to oci
func (p *Provider) CreateVolume(ctx *lepton.Context, name, data, provider string) (vol lepton.NanosVolume, err error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")

	size, err := lepton.GetSizeInGb(ctx.Config().BaseVolumeSz)
	if err != nil {
		ctx.Logger().Log(err.Error())
		err = errors.New("failed converting size to number")
		return
	}
	sizeInGBs := int64(size)

	if sizeInGBs < 50 {
		sizeInGBs = 50
	}

	createVolumeRes, err := p.blockstorageClient.CreateVolume(context.TODO(), core.CreateVolumeRequest{
		CreateVolumeDetails: core.CreateVolumeDetails{
			AvailabilityDomain: &p.availabilityDomain,
			CompartmentId:      &p.compartmentID,
			DisplayName:        &name,
			FreeformTags:       ociOpsTags,
			SizeInGBs:          &sizeInGBs,
		},
	})
	if err != nil {
		return
	}

	vol = lepton.NanosVolume{
		ID:   *createVolumeRes.Id,
		Name: name,
		Size: strconv.Itoa(int(sizeInGBs)),
	}

	return
}

// GetAllVolumes returns a list of oci volumes
func (p *Provider) GetAllVolumes(ctx *lepton.Context) (vols *[]lepton.NanosVolume, err error) {
	return nil, errors.New("Unsupported")

	listVolumesResponse, err := p.blockstorageClient.ListVolumes(context.TODO(), core.ListVolumesRequest{CompartmentId: &p.compartmentID})
	if err != nil {
		return nil, err
	}

	vols = &[]lepton.NanosVolume{}

	for _, v := range listVolumesResponse.Items {
		if checkHasOpsTags(v.FreeformTags) {
			*vols = append(*vols, lepton.NanosVolume{
				ID:        *v.Id,
				Name:      *v.DisplayName,
				Status:    string(v.LifecycleState),
				Size:      strconv.Itoa(int(*v.SizeInGBs)),
				CreatedAt: lepton.Time2Human(v.TimeCreated.Time),
			})
		}
	}

	return
}

func (p *Provider) getVolumeByName(ctx *lepton.Context, name string) (vol *lepton.NanosVolume, err error) {
	return nil, errors.New("Unsupported")

	vols, err := p.GetAllVolumes(ctx)
	if err != nil {
		return
	}

	for _, v := range *vols {
		if v.Name == name {
			vol = &v
			return
		}
	}

	err = errors.New("volume not found")
	return
}

// DeleteVolume removes an oci volume
func (p *Provider) DeleteVolume(ctx *lepton.Context, name string) (err error) {
	return errors.New("Unsupported")

	vol, err := p.getVolumeByName(ctx, name)
	if err != nil {
		return
	}

	_, err = p.blockstorageClient.DeleteVolume(context.TODO(), core.DeleteVolumeRequest{VolumeId: &vol.ID})

	return
}

// AttachVolume attaches a volume to an oci instance
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name string, attachID int) (err error) {
	return errors.New("Unsupported")

	_, err = p.computeClient.AttachVolume(context.TODO(), core.AttachVolumeRequest{
		AttachVolumeDetails: core.AttachParavirtualizedVolumeDetails{},
	})

	return
}

// DetachVolume detaches a volume from an oci instance
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) (err error) {
	return errors.New("Unsupported")

	_, err = p.computeClient.DetachVolume(context.TODO(), core.DetachVolumeRequest{})

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
