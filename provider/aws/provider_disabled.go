//go:build !aws && onlyprovider

package aws

import (
	"github.com/nanovms/ops/provider/disabled"
)

// ProviderName ...
const ProviderName = "aws"

// NewProvider ...
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
