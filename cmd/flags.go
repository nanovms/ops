package cmd

import (
	"github.com/nanovms/ops/types"
)

// MergeConfigFlags are flags structures able to override ops configuration attributes
type MergeConfigFlags interface {
	MergeToConfig(config *types.Config) error
}

// MergeConfigContainer is responsible for merge a list of flags attributes to ops configuration
type MergeConfigContainer struct {
	flags []MergeConfigFlags
}

// NewMergeConfigContainer returns an instance of MergeConfigContainer
// Flags order matters.
func NewMergeConfigContainer(flags ...MergeConfigFlags) *MergeConfigContainer {
	return &MergeConfigContainer{flags}
}

// Merge uses a list of flags to override configuration properties.
func (m *MergeConfigContainer) Merge(config *types.Config) error {

	for _, f := range m.flags {
		err := f.MergeToConfig(config)
		if err != nil {
			return err
		}
	}

	return nil
}
