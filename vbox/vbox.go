package vbox

import "github.com/nanovms/ops/types"

// Provider provides access to the VirtualBox API.
type Provider struct{}

// NewProvider returns an instance of VirtualBox Provider
func NewProvider() *Provider {
	return &Provider{}
}

// Initialize checks conditions to use VirtualBox
func (p *Provider) Initialize(c *types.ProviderConfig) error {
	return nil
}
