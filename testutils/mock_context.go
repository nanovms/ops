package testutils

import (
	"github.com/nanovms/ops/lepton"
)

// NewMockContext returns a context mock
func NewMockContext() *lepton.Context {
	return lepton.NewContext(lepton.NewConfig())
}
