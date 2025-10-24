//go:build scaleway || !onlyprovider

package scaleway

import (
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ProviderName of the cloud platform provider
const ProviderName = "scaleway"

// Scaleway Provider to interact with Scaleway cloud infrastructure
type Scaleway struct {
	Storage *ObjectStorage
	client  *scw.Client
}

// NewProvider Scaleway
func NewProvider() *Scaleway {
	return &Scaleway{}
}

// Initialize Scaleway client
func (h *Scaleway) Initialize(c *types.ProviderConfig) error {
	var err error

	accessKeyID := os.Getenv("SCALEWAY_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("SCALEWAY_SECRET_ACCESS_KEY")

	h.client, err = scw.NewClient(
		scw.WithAuth(accessKeyID, secretAccessKey),
		scw.WithDefaultOrganizationID(os.Getenv("SCALEWAY_ORGANIZATION_ID")),
		scw.WithDefaultZone(scw.Zone(c.Zone)),
	)

	h.Storage = &ObjectStorage{}
	return err
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
