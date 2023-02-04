package digitalocean

import "github.com/nanovms/ops/lepton"

// CreateVolume is a stub to satisfy VolumeService interface
func (do *DigitalOcean) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	var vol lepton.NanosVolume
	return vol, nil
}

// GetAllVolumes is a stub to satisfy VolumeService interface
func (do *DigitalOcean) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume is a stub to satisfy VolumeService interface
func (do *DigitalOcean) DeleteVolume(ctx *lepton.Context, name string) error {
	return nil
}

// AttachVolume is a stub to satisfy VolumeService interface
func (do *DigitalOcean) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	return nil
}

// DetachVolume is a stub to satisfy VolumeService interface
func (do *DigitalOcean) DetachVolume(ctx *lepton.Context, image, name string) error {
	return nil
}
