//go:build linode || !onlyprovider

package linode

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "linode"

// Linode Provider to interact with Linode infrastructure
type Linode struct {
	Storage *Objects
	token   string
}

// NewProvider Linode
func NewProvider() *Linode {
	return &Linode{}
}

// Initialize provider
func (v *Linode) Initialize(config *types.ProviderConfig) error {
	v.token = os.Getenv("TOKEN")
	if v.token == "" {
		return fmt.Errorf("TOKEN is not set")
	}

	//	ctx := context.Background()
	return nil
}

// GetStorage returns storage interface for cloud provider
func (v *Linode) GetStorage() lepton.Storage {
	return v.Storage
}
