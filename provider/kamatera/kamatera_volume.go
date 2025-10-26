package kamatera

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// CreateVolume is currently a stub for Kamatera volume creation.
func (*Kamatera) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil // Empty implementation
}

// GetAllVolumes returns an empty list because Kamatera volumes are not supported yet.
func (*Kamatera) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	// Returns an empty slice of NanosVolume
	return &[]lepton.NanosVolume{}, nil // Empty implementation
}

// DeleteVolume is a stub; Kamatera volume deletion is not implemented.
func (*Kamatera) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil // Empty implementation
}

// AttachVolume is a stub because Kamatera volume attachment is not implemented.
func (*Kamatera) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil // Empty implementation
}

// DetachVolume is a stub because Kamatera volume detachment is not implemented.
func (*Kamatera) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil // Empty implementation
}
