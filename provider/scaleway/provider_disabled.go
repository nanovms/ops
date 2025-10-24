//go:build !scaleway && onlyprovider

package scaleway

import "github.com/nanovms/ops/provider/disabled"

// ProviderName of the cloud platform provider
const ProviderName = "scaleway"

// NewProvider Scaleway
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
