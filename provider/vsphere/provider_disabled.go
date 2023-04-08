//go:build !vsphere && onlyprovider

package vsphere

import (
	"github.com/nanovms/ops/provider/disabled"
)

// ProviderName ...
const ProviderName = "vsphere"

// NewProvider ...
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
