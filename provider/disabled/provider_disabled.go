//go:build !(aws && azure && (digitalocean || do) && gcp && hyperv && ibm && linode && oci && openshift && openstack && proxmox && upcloud && vbox && vsphere && vultr) && onlyprovider

package disabled

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// Provider ...
type Provider struct {
	name string
}

// NewProvider ...
func NewProvider(name string) *Provider {
	return &Provider{name}
}

// Initialize ...
func (p *Provider) Initialize(config *types.ProviderConfig) error {
	return fmt.Errorf("[%s] provider - disabled", p.name)
}

// BuildImage ...
func (p *Provider) BuildImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// BuildImageWithPackage ...
func (p *Provider) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	return "", nil
}

// CreateImage ...
func (p *Provider) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// ListImages ...
func (p *Provider) ListImages(ctx *lepton.Context) error {
	return nil
}

// GetImages ...
func (p *Provider) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}

// DeleteImage ...
func (p *Provider) DeleteImage(ctx *lepton.Context, imagename string) error {
	return nil
}

// ResizeImage ...
func (p *Provider) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return nil
}

// SyncImage ...
func (p *Provider) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return nil
}

// CustomizeImage ...
func (p *Provider) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// CreateInstance ...
func (p *Provider) CreateInstance(ctx *lepton.Context) error {
	return nil
}

// ListInstances ...
func (p *Provider) ListInstances(ctx *lepton.Context) error {
	return nil
}

// GetInstances ...
func (p *Provider) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstanceByName ...
func (p *Provider) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// DeleteInstance ...
func (p *Provider) DeleteInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StopInstance ...
func (p *Provider) StopInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StartInstance ...
func (p *Provider) StartInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// RebootInstance ...
func (p *Provider) RebootInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// GetInstanceLogs ...
func (p *Provider) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// PrintInstanceLogs ...
func (p *Provider) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// CreateVolume ...
func (p *Provider) CreateVolume(ctx *lepton.Context, volumeName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil
}

// GetAllVolumes ...
func (p *Provider) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume ...
func (p *Provider) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil
}

// AttachVolume ...
func (p *Provider) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil
}

// DetachVolume ...
func (p *Provider) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil
}

// CreateVolumeImage ...
func (p *Provider) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil
}

// CreateVolumeFromSource ...
func (p *Provider) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return nil
}
