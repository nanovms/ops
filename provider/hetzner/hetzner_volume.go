package hetzner

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// CreateVolume is currently a stub for Hetzner volume creation.
func (*Hetzner) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil // Empty implementation
}

// GetAllVolumes returns an empty list because Hetzner volumes are not supported yet.
func (*Hetzner) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	// Returns an empty slice of NanosVolume
	return &[]lepton.NanosVolume{}, nil // Empty implementation
}

// DeleteVolume is a stub; Hetzner volume deletion is not implemented.
func (*Hetzner) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil // Empty implementation
}

// AttachVolume is a stub because Hetzner volume attachment is not implemented.
func (*Hetzner) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil // Empty implementation
}

// DetachVolume is a stub because Hetzner volume detachment is not implemented.
func (*Hetzner) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil // Empty implementation
}
