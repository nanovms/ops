package bhyve

import (
	"github.com/nanovms/ops/lepton"
)

// CreateVolume creates local volume 
func (b *Bhyve) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	var vol lepton.NanosVolume
	return vol, nil
}

// GetAllVolumes gets all volumes created in bhyve
func (b *Bhyve) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume deletes specific disk and image in bhyve
func (b *Bhyve) DeleteVolume(ctx *lepton.Context, name string) error {
	return nil
}

// AttachVolume attaches volume to existing instance
func (b *Bhyve) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	return nil
}

// DetachVolume detaches volume from existing instance
func (b *Bhyve) DetachVolume(ctx *lepton.Context, image, volumeName string) error {
	return nil
}
