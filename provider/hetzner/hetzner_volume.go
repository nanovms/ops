package hetzner

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

func (*Hetzner) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil // Empty implementation
}

func (*Hetzner) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	// Returns an empty slice of NanosVolume
	return &[]lepton.NanosVolume{}, nil // Empty implementation
}

func (*Hetzner) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil // Empty implementation
}

func (*Hetzner) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil // Empty implementation
}

func (*Hetzner) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil // Empty implementation
}
