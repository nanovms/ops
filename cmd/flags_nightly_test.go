package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateNightlyFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistNightlyCommandFlags(flagSet)

	flagSet.Set("nightly", "true")

	nightlyFlags := cmd.NewNightlyCommandFlags(flagSet)

	assert.Equal(t, nightlyFlags.Nightly, true)
}

func TestNightlyFlagsMergeToConfig(t *testing.T) {
	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistNightlyCommandFlags(flagSet)

	flagSet.Set("nightly", "true")

	nightlyFlags := cmd.NewNightlyCommandFlags(flagSet)

	opsPath := lepton.GetOpsHome() + "/nightly"

	c := &types.Config{}
	expected := &types.Config{
		Boot:         opsPath + "/boot.img",
		Kernel:       opsPath + "/kernel.img",
		CloudConfig:  types.ProviderConfig{},
		RunConfig:    types.RunConfig{},
		NameServer:   "8.8.8.8",
		NightlyBuild: true,
	}

	nightlyFlags.MergeToConfig(c)

	assert.Equal(t, expected, c)
}
