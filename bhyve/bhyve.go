package bhyve

import (
	"github.com/nanovms/ops/types"
)

// Bhyve specific config
type Bhyve struct {
}


func (b *Bhyve) Initialize(config *types.ProviderConfig) error {
	return nil
}
