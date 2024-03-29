//go:build vultr || !onlyprovider

package vultr

import (
	"context"
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

// ProviderName of the cloud platform provider
const ProviderName = "vultr"

func stripZone(zone string) string {
	return zone[0:3]
}

// Vultr Provider to interact with Vultr infrastructure
type Vultr struct {
	Storage *Objects
	Client  *govultr.Client
}

// NewProvider Vultr
func NewProvider() *Vultr {
	return &Vultr{}
}

// Initialize provider
func (v *Vultr) Initialize(config *types.ProviderConfig) error {
	apiKey := os.Getenv("VULTR_TOKEN")
	if apiKey == "" {
		return fmt.Errorf("VULTR_TOKEN is not set")
	}

	vultrConfig := &oauth2.Config{}
	ctx := context.Background()
	ts := vultrConfig.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	v.Client = govultr.NewClient(oauth2.NewClient(ctx, ts))
	return nil
}

// GetStorage returns storage interface for cloud provider
func (v *Vultr) GetStorage() lepton.Storage {
	return v.Storage
}
