package scaleway

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// CreateVolume is currently a stub for Scaleway volume creation.
func (*Scaleway) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil // Empty implementation
}

// GetAllVolumes returns an empty list because Scaleway volumes are not supported yet.
func (*Scaleway) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	// Returns an empty slice of NanosVolume
	return &[]lepton.NanosVolume{}, nil // Empty implementation
}

// DeleteVolume is a stub; Scaleway volume deletion is not implemented.
func (*Scaleway) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil // Empty implementation
}

// AttachVolume is a stub because Scaleway volume attachment is not implemented.
func (*Scaleway) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil // Empty implementation
}

// DetachVolume is a stub because Scaleway volume detachment is not implemented.
func (*Scaleway) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil // Empty implementation
}
