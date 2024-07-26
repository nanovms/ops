//go:build vbox || !onlyprovider

package vbox

import (
	"errors"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// CreateVolume creates a local volume and uploads the volume to upcloud
func (p *Provider) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	vol, err := lepton.CreateLocalVolume(ctx.Config(), cv.Name, data, provider)
	if err != nil {
		return vol, nil
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
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name string, attachID int) (err error) {
	err = errors.New("Unsupported")

	return
}

// DetachVolume detaches a storage from an upcloud server
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) (err error) {
	err = errors.New("Unsupported")

	return
}
