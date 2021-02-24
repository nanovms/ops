package hyperv

import (
	"errors"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// Provider provides access to the Hyper-V API.
type Provider struct {
}

// NewProvider returns an instance of Hyper-V provider
func NewProvider() *Provider {
	return &Provider{}
}

// Initialize checks conditions to use hyper-v
func (p *Provider) Initialize(c *types.ProviderConfig) error {
	if available, _, _ := IsPowershellAvailable(); !available {
		return errors.New("powershell not available")
	}

	isAdmin, err := isCurrentUserAnAdministrator()
	if err != nil {
		return err
	} else if !isAdmin {
		return errors.New("this feature is only supported on terminals with elevated privileges")
	}

	return nil
}

// CreateVolume is a stub
func (p *Provider) CreateVolume(ctx *lepton.Context, name, data, size, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// GetAllVolumes is a stub
func (p *Provider) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, errors.New("Unsupported")
}

// DeleteVolume is a stub
func (p *Provider) DeleteVolume(ctx *lepton.Context, name string) error {
	return errors.New("Unsupported")
}

// AttachVolume is a stub
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name, mount string) error {
	return errors.New("Unsupported")
}

// DetachVolume is a stub
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) error {
	return errors.New("Unsupported")
}
