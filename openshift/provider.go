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

func NewProvider() *Openshift {

	return &Openshift{}
}

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

func (oc *Openshift) BuildImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

func (oc *Openshift) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	return "", nil
}

func (oc *Openshift) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

func (oc *Openshift) ListImages(ctx *lepton.Context) error {
	return nil
}
func (oc *Openshift) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}
func (oc *Openshift) DeleteImage(ctx *lepton.Context, imagename string) error {
	return nil
}
func (oc *Openshift) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return nil
}
func (oc *Openshift) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return nil
}
func (oc *Openshift) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

func (oc *Openshift) CreateInstance(ctx *lepton.Context) error {
	return nil
}
func (oc *Openshift) ListInstances(ctx *lepton.Context) error {
	return nil
}
func (oc *Openshift) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, nil
}

func (oc *Openshift) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

func (oc *Openshift) DeleteInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

func (oc *Openshift) StopInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

func (oc *Openshift) StartInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

func (oc *Openshift) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

func (oc *Openshift) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

func (oc *Openshift) CreateVolume(ctx *lepton.Context, volumeName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, nil
}

func (oc *Openshift) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	return nil, nil
}

func (oc *Openshift) DeleteVolume(ctx *lepton.Context, volumeName string) error {
	return nil
}
func (oc *Openshift) AttachVolume(ctx *lepton.Context, instanceName, volumeName string, attachID int) error {
	return nil
}
func (oc *Openshift) DetachVolume(ctx *lepton.Context, instanceName, volumeName string) error {
	return nil
}
