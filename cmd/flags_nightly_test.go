package cmd_test

import (
	"testing"

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

	config := &lepton.Config{}
	expected := &lepton.Config{
		Boot:         opsPath + "/boot.img",
		Kernel:       opsPath + "/kernel.img",
		CloudConfig:  lepton.ProviderConfig{},
		RunConfig:    lepton.RunConfig{},
		NameServer:   "8.8.8.8",
		NightlyBuild: true,
	}

	nightlyFlags.MergeToConfig(config)

	assert.Equal(t, expected, config)
}
