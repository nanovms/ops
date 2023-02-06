package onprem

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "onprem"

// OnPrem Provider to interact with OnPrem infrastructure
type OnPrem struct{}

// NewProvider OnPrem
func NewProvider() *OnPrem {
	return &OnPrem{}
}

// Initialize on prem provider
func (p *OnPrem) Initialize(config *types.ProviderConfig) error {
	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *OnPrem) GetStorage() lepton.Storage {
	return nil
}
