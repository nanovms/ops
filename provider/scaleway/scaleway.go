//go:build scaleway || !onlyprovider

package scaleway

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "scaleway"

const (
	opsLabelKey          = "createdBy"
	opsLabelValue        = "ops"
	opsImageNameLabelKey = "opsImageName"
	opsInstanceLabelKey  = "opsInstance"
	opsImageBuilderLabel = "opsImageBuilder"
)

// Scaleway Provider to interact with Scaleway cloud infrastructure
type Scaleway struct {
	Storage *ObjectStorage
}

// NewProvider Scaleway
func NewProvider() *Scaleway {
	return &Scaleway{}
}

// Initialize Scaleway client
func (h *Scaleway) Initialize(c *types.ProviderConfig) error {
	h.Storage = &ObjectStorage{}
	return nil
}

// GetStorage returns storage interface for cloud provider
func (h *Scaleway) GetStorage() lepton.Storage {
	return h.Storage
}

func (h *Scaleway) ensureStorage() {
	if h.Storage == nil {
		h.Storage = &ObjectStorage{}
	}
}

func managedLabelSelector() string {
	return fmt.Sprintf("%s=%s", opsLabelKey, opsLabelValue)
}
