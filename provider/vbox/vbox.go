//go:build vbox || !onlyprovider

package vbox

import "github.com/nanovms/ops/types"

// ProviderName of the cloud platform provider
const ProviderName = "vbox"

// Provider to interact with VirtualBox infrastructure
type Provider struct{}

// NewProvider VirtualBox
func NewProvider() *Provider {
	return &Provider{}
}

// Initialize checks conditions to use VirtualBox
func (p *Provider) Initialize(c *types.ProviderConfig) error {
	return nil
}
