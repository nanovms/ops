package digitalocean

import (
	"os"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// DigitalOcean provides access to the DigitalOcean API.
type DigitalOcean struct {
	Storage *Spaces
	Client  *godo.Client
}

// Initialize DigialOcean related things
func (do *DigitalOcean) Initialize(c *types.ProviderConfig) error {
	doToken := os.Getenv("DO_TOKEN")
	do.Client = godo.NewFromToken(doToken)
	return nil
}

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() lepton.Storage {
	return do.Storage
}
