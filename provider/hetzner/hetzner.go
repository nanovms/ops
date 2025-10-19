//go:build hetzner || !onlyprovider

package hetzner

import (
	"fmt"
	"os"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "hetzner"

const (
	opsLabelKey          = "createdBy"
	opsLabelValue        = "ops"
	opsImageNameLabelKey = "opsImageName"
	opsInstanceLabelKey  = "opsInstance"
	opsImageBuilderLabel = "opsImageBuilder"
	defaultServerType    = "cx23"
	defaultBuilderImage  = "ubuntu-22.04"
)

// Hetzner Provider to interact with Hetzner cloud infrastructure
type Hetzner struct {
	Client  *hcloud.Client
	Storage *ObjectStorage
}

// NewProvider Hetzner
func NewProvider() *Hetzner {
	return &Hetzner{}
}

// Initialize Hetzner client
func (h *Hetzner) Initialize(c *types.ProviderConfig) error {
	hetznerToken := os.Getenv("HCLOUD_TOKEN")
	if hetznerToken == "" {
		return fmt.Errorf("set HCLOUD_TOKEN")
	}
	h.Client = hcloud.NewClient(hcloud.WithToken(hetznerToken))
	h.Storage = &ObjectStorage{}
	return nil
}

// GetStorage returns storage interface for cloud provider
func (h *Hetzner) GetStorage() lepton.Storage {
	return h.Storage
}

func (h *Hetzner) ensureStorage() {
	if h.Storage == nil {
		h.Storage = &ObjectStorage{}
	}
}

func managedLabelSelector() string {
	return fmt.Sprintf("%s=%s", opsLabelKey, opsLabelValue)
}
