package digitalocean

import (
	"os"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/config"
	"github.com/nanovms/ops/lepton"
)

// DigitalOcean provides access to the DigitalOcean API.
type DigitalOcean struct {
	Storage *Spaces
	Client  *godo.Client
}

// Initialize DigialOcean related things
func (do *DigitalOcean) Initialize(c *config.ProviderConfig) error {
	doToken := os.Getenv("TOKEN")
	do.Client = godo.NewFromToken(doToken)
	return nil
}

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() lepton.Storage {
	return do.Storage
}
