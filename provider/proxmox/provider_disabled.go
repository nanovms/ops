//go:build !proxmox && onlyprovider

package proxmox

import (
	"github.com/nanovms/ops/provider/disabled"
)

// ProviderName ...
const ProviderName = "proxmox"

// NewProvider ...
func NewProvider() *disabled.Provider {
	return disabled.NewProvider(ProviderName)
}
