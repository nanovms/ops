//go:build linode || !onlyprovider

package linode

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Linode) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	var vol lepton.NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Linode) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Linode) DeleteVolume(ctx *lepton.Context, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Linode) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Linode) DetachVolume(ctx *lepton.Context, image, name string) error {
	return nil
}
