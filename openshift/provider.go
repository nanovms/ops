package openshift

import (
	"fmt"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// Openshift provides access to the Openshift/Kubernetes API and implements the provider interface.
type Openshift struct {
	client *Client
}

// NewProvider returns an instance of Openshift provider
func NewProvider() *Openshift {

	return &Openshift{}
}

// Initialize prepares the openshift client and checks if the openshift server is up
func (oc *Openshift) Initialize(config *types.ProviderConfig) error {
	client, err := New()
	if err != nil {
		return err
	}
	up, err := client.IsServerUp(10 * time.Second)
	if err != nil {
		return err
	}
	if !up {
		return fmt.Errorf("openshift cluster is not up/available")
	}

	oc.client = client
	return nil
}

// BuildImage builds the image
func (oc *Openshift) BuildImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// BuildImageWithPackage builds the image with package
func (oc *Openshift) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	return "", nil
}

// CreateImage creates the image
func (oc *Openshift) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// ListImages lists all the images
func (oc *Openshift) ListImages(ctx *lepton.Context) error {
	return nil
}

// GetImages gets all the images
func (oc *Openshift) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}

// DeleteImage deletes the image using the image name
func (oc *Openshift) DeleteImage(ctx *lepton.Context, imagename string) error {
	return nil
}

// ResizeImage resizes the image
func (oc *Openshift) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return nil
}

// SyncImage sync an image from one provider to another
func (oc *Openshift) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return nil
}

// CustomizeImage customizes the images
func (oc *Openshift) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// CreateInstance creates nano instance
func (oc *Openshift) CreateInstance(ctx *lepton.Context) error {
	return nil
}

// ListInstances lists all nano instances locally or on the provider
func (oc *Openshift) ListInstances(ctx *lepton.Context) error {
	return nil
}

// GetInstances gets all nano instances
func (oc *Openshift) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstanceByName gets a nano instance by name
func (oc *Openshift) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// DeleteInstance deletes a nano instance using the name
func (oc *Openshift) DeleteInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StopInstance stops a nanos instance
func (oc *Openshift) StopInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StartInstance starts a nanos instance
func (oc *Openshift) StartInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// GetInstanceLogs gets the logs from a nanos instance
func (oc *Openshift) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// PrintInstanceLogs prints the logs from a nanos instance
func (oc *Openshift) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// CreateVolume creates a volume on the openshift cluster
func (oc *Openshift) CreateVolume(ctx *lepton.Context, volumeName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil
}

// GetAllVolumes gets all volumes
func (oc *Openshift) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume deletes a volume
func (oc *Openshift) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil
}

// AttachVolume attaches a volume to a nano instance
func (oc *Openshift) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil
}

// DetachVolume detaches a volume from a nano instance
func (oc *Openshift) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil
}
