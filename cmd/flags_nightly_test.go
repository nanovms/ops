package cmd

import (
	"path"
	"testing"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateNightlyFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	PersistNightlyCommandFlags(flagSet)

	flagSet.Set("nightly", "true")

	nightlyFlags := NewNightlyCommandFlags(flagSet)

	assert.Equal(t, nightlyFlags.Nightly, true)
}

func TestNightlyFlagsMergeToConfig(t *testing.T) {

	nightlyPath := path.Join(lepton.GetOpsHome(), "nightly")

	t.Run("if nighly flag is enabled should set boot and kernel nightly paths ", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistNightlyCommandFlags(flagSet)

		flagSet.Set("nightly", "true")

		nightlyFlags := NewNightlyCommandFlags(flagSet)

		c := &types.Config{}
		expected := &types.Config{
			Boot:         path.Join(nightlyPath, "boot.img"),
			UefiBoot:     path.Join(nightlyPath, "bootx64.efi"),
			Kernel:       path.Join(nightlyPath, "/kernel.img"),
			CloudConfig:  types.ProviderConfig{},
			RunConfig:    types.RunConfig{},
			NameServers:  []string{"8.8.8.8"},
			NightlyBuild: true,
		}

		nightlyFlags.MergeToConfig(c)

		assert.Equal(t, expected, c)
	})
}
