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

// relayered Provider to interact with relayered infrastructure
type relayered struct {
	Storage *Objects
	token   string
	iam     string
}

// NewProvider relayered
func NewProvider() *relayered {
	return &relayered{}
}

// Initialize provider
func (v *relayered) Initialize(config *types.ProviderConfig) error {
	v.token = os.Getenv("TOKEN")
	if v.token == "" {
		return fmt.Errorf("TOKEN is not set")
	}

	return nil
}

// Token is the return type for a new IAM token.
type Token struct {
	AccessToken string `json:"access_token"`
}

// GetStorage returns storage interface for cloud provider
func (v *relayered) GetStorage() lepton.Storage {
	return v.Storage
}
