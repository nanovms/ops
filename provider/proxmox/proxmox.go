//go:build proxmox || !onlyprovider

package proxmox

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "proxmox"

// ProxMox provides access to the ProxMox API.
type ProxMox struct {
	Storage  *Objects
	tokenID  string
	secret   string
	apiURL   string
	nodeNAME string

	// many of these belong in their own structs
	// {Image, Instance, etc..}

	// arch specifies the type of CPU architecture
	arch string `cloud:"arch"`

	// cores of CPU
	cores string `cloud:"cores"`

	// machine specifies the type of machine
	machine string `cloud:"machine"`

	// memory
	memory string `cloud:"memory"`

	// numa
	numa string `cloud:"numa"`

	// instanceName
	instanceName string `cloud:"instancename"`

	// imageName
	imageName string `cloud:"imagename"`

	// bridgePrefix - prefix for network bridge interfaces on bare metal host (like br/vmbr/bridge)
	bridgePrefix string `cloud:"bridgeprefix"`

	// isoStorageName is used for upload intermediate iso images via ProxMox API
	isoStorageName string `cloud:"isostoragename"`

	// onboot is used to define automatic startup option for instance
	onboot string `cloud:"onboot"`

	// protection is used to define vm/image protection for instance
	protection string `cloud:"protection"`

	// sockets of CPUs
	sockets string `cloud:"sockets"`

	// storageName is used for create bootable raw image for instance via ProxMox API from iso image
	storageName string `cloud:"storagename"`
}

// NewProvider ProxMox
func NewProvider() *ProxMox {
	return &ProxMox{}
}

// Initialize provider
func (p *ProxMox) Initialize(config *types.ProviderConfig) error {

	var err error

	p.tokenID = os.Getenv("TOKEN_ID")
	if p.tokenID == "" {
		return fmt.Errorf("TOKEN_ID is not set")
	}

	p.secret = os.Getenv("SECRET")
	if p.secret == "" {
		return fmt.Errorf("SECRET is not set")
	}

	p.apiURL = os.Getenv("API_URL")
	if p.apiURL == "" {
		return fmt.Errorf("API_URL is not set")
	}

	p.nodeNAME = os.Getenv("NODE_NAME")
	if p.nodeNAME == "" {
		p.nodeNAME = "pve"
	}

	err = p.CheckInit()
	if err != nil {
		return err
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *ProxMox) GetStorage() lepton.Storage {
	return p.Storage
}
