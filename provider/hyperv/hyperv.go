//go:build hyperv || !onlyprovider

package hyperv

import (
	"errors"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/nanovms/ops/wsl"
)

// ProviderName of the cloud platform provider
const ProviderName = "hyper-v"

// Provider to interact with Hyper-V cloud infrastructure
type Provider struct{}

// NewProvider Hyper-V
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

	if !wsl.IsWSL() {
		return errors.New("Hyper-v is only supported on WSL")
	}

	return nil
}

// CreateVolume is a stub
func (p *Provider) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
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
func (p *Provider) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	return errors.New("Unsupported")
}

// DetachVolume is a stub
func (p *Provider) DetachVolume(ctx *lepton.Context, image, name string) error {
	return errors.New("Unsupported")
}

// CreateVolumeImage ...
func (p *Provider) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (p *Provider) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
