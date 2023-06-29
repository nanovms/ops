//go:build relayered || !onlyprovider

package relayered

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "relayered"

// Relayered Provider to interact with relayered infrastructure
type Relayered struct {
	Storage *Objects
	token   string
}

// NewProvider relayered
func NewProvider() *Relayered {
	return &Relayered{}
}

// Initialize provider
func (v *Relayered) Initialize(config *types.ProviderConfig) error {
	v.token = os.Getenv("RELAYERED_TOKEN")
	if v.token == "" {
		return fmt.Errorf("RELAYERED_TOKEN is not set")
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (v *Relayered) GetStorage() lepton.Storage {
	return v.Storage
}
