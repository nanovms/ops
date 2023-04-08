//go:build !ibm && onlyprovider

package ibm

import (
	"github.com/nanovms/ops/provider/disabled"
)

// ProviderName ...
const ProviderName = "ibm"

// NewProvider ...
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
