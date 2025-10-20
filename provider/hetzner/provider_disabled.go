//go:build !hetzner && onlyprovider

package hetzner

import "github.com/nanovms/ops/provider/disabled"

// ProviderName of the cloud platform provider
const ProviderName = "hetzner"

// NewProvider Hetzner
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
