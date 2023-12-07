//go:build openshift || !onlyprovider

package openshift

import (
	"errors"
	"fmt"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "openshift"

// OpenShift provides access to the OpenShift/Kubernetes API and implements the provider interface.
type OpenShift struct {
	client *Client
}

// NewProvider Openshift
func NewProvider() *OpenShift {
	return &OpenShift{}
}

// Initialize prepares the openshift client and checks if the openshift server is up
func (oc *OpenShift) Initialize(config *types.ProviderConfig) error {
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
func (oc *OpenShift) BuildImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// BuildImageWithPackage builds the image with package
func (oc *OpenShift) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	return "", nil
}

// CreateImage creates the image
func (oc *OpenShift) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// ListImages lists all the images
func (oc *OpenShift) ListImages(ctx *lepton.Context) error {
	return nil
}

// GetImages gets all the images
func (oc *OpenShift) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}

// DeleteImage deletes the image using the image name
func (oc *OpenShift) DeleteImage(ctx *lepton.Context, imagename string) error {
	return nil
}

// ResizeImage resizes the image
func (oc *OpenShift) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return nil
}

// SyncImage sync an image from one provider to another
func (oc *OpenShift) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return nil
}

// CustomizeImage customizes the images
func (oc *OpenShift) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// CreateInstance creates nano instance
func (oc *OpenShift) CreateInstance(ctx *lepton.Context) error {
	return nil
}

// ListInstances lists all nano instances locally or on the provider
func (oc *OpenShift) ListInstances(ctx *lepton.Context) error {
	return nil
}

// GetInstances gets all nano instances
func (oc *OpenShift) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstanceByName gets a nano instance by name
func (oc *OpenShift) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// DeleteInstance deletes a nano instance using the name
func (oc *OpenShift) DeleteInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StopInstance stops a nanos instance
func (oc *OpenShift) StopInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// RebootInstance reboots a nanos instance.
func (oc *OpenShift) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts a nanos instance
func (oc *OpenShift) StartInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// GetInstanceLogs gets the logs from a nanos instance
func (oc *OpenShift) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// PrintInstanceLogs prints the logs from a nanos instance
func (oc *OpenShift) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// CreateVolume creates a volume on the openshift cluster
func (oc *OpenShift) CreateVolume(ctx *lepton.Context, volumeName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil
}

// GetAllVolumes gets all volumes
func (oc *OpenShift) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

// DeleteVolume deletes a volume
func (oc *OpenShift) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil
}

// AttachVolume attaches a volume to a nano instance
func (oc *OpenShift) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil
}

// DetachVolume detaches a volume from a nano instance
func (oc *OpenShift) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil
}

// CreateVolumeImage ...
func (oc *OpenShift) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (oc *OpenShift) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
