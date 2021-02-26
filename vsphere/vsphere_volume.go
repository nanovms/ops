package vsphere

import "github.com/nanovms/ops/lepton"

// CreateVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) CreateVolume(ctx *lepton.Context, name, data, size, provider string) (lepton.NanosVolume, error) {
	var vol lepton.NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (v *Vsphere) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DeleteVolume(ctx *lepton.Context, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) AttachVolume(ctx *lepton.Context, image, name, mount string) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (v *Vsphere) DetachVolume(ctx *lepton.Context, image, name string) error {
	return nil
}
