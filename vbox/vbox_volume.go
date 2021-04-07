package vbox

import (
	"errors"
	"os"

	"github.com/nanovms/ops/lepton"
)

// CreateVolume creates a local volume and uploads the volume to upcloud
func (p *Provider) CreateVolume(ctx *lepton.Context, name, data, size, provider string) (vol lepton.NanosVolume, err error) {
	vol, err = lepton.CreateLocalVolume(ctx.Config(), name, data, size, provider)
	if err != nil {
		return
	}
	defer os.Remove(vol.Path)

	return vol, nil
}

// GetAllVolumes returns every upcloud volume
func (p *Provider) GetAllVolumes(ctx *lepton.Context) (volumes *[]lepton.NanosVolume, err error) {
	volumes = &[]lepton.NanosVolume{}

	err = errors.New("Unsupported")

	return
}

// DeleteVolume deletes a volume from upcloud
func (p *Provider) DeleteVolume(ctx *lepton.Context, name string) (err error) {
	err = errors.New("Unsupported")

	return
}

// AttachVolume attaches a storage to an upcloud server
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name string) (err error) {
	err = errors.New("Unsupported")

	return
}

// DetachVolume detaches a storage from an upcloud server
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) (err error) {
	err = errors.New("Unsupported")

	return
}
