//go:build digitalocean || do || !onlyprovider

package digitalocean

import (
	"fmt"
	"os"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "do"

// DigitalOcean Provider to interact with DigitalOcean cloud infrastructure
type DigitalOcean struct {
	Storage *Spaces
	Client  *godo.Client
}

// NewProvider DigitalOcean
func NewProvider() *DigitalOcean {
	return &DigitalOcean{}
}

// Initialize DigialOcean related things
func (do *DigitalOcean) Initialize(c *types.ProviderConfig) error {
	doToken := os.Getenv("DO_TOKEN")
	if doToken == "" {
		return fmt.Errorf("set DO_TOKEN")
	}
	do.Client = godo.NewFromToken(doToken)
	return nil
}

// GetStorage returns storage interface for cloud provider
func (do *DigitalOcean) GetStorage() lepton.Storage {
	return do.Storage
}
