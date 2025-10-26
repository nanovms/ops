//go:build !kamatera && onlyprovider

package kamatera

import "github.com/nanovms/ops/provider/disabled"

// ProviderName of the cloud platform provider
const ProviderName = "kamatera"

// NewProvider kamatera
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
