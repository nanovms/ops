package onprem

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName is the name of the cloud platform provider.
const ProviderName = "onprem"

// OnPrem provider for ops
type OnPrem struct{}

// Initialize on prem provider
func (p *OnPrem) Initialize(config *types.ProviderConfig) error {
	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *OnPrem) GetStorage() lepton.Storage {
	return nil
}
