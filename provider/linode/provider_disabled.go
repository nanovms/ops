//go:build !linode && onlyprovider

package linode

import (
	"github.com/nanovms/ops/provider/disabled"
)

// ProviderName ...
const ProviderName = "linode"

// NewProvider ...
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
