package cmd_test

import (
	"path"
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

	nightlyPath := path.Join(lepton.GetOpsHome(), "nightly")
	currentOpsPath := path.Join(lepton.GetOpsHome(), lepton.LocalReleaseVersion)

	t.Run("if nighly flag is enabled should set boot and kernel nightly paths ", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistNightlyCommandFlags(flagSet)

		flagSet.Set("nightly", "true")

		nightlyFlags := cmd.NewNightlyCommandFlags(flagSet)

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

	t.Run("if nighly flag is enabled should override current configuration boot and kernel paths with nightly paths", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistNightlyCommandFlags(flagSet)

		flagSet.Set("nightly", "true")

		nightlyFlags := cmd.NewNightlyCommandFlags(flagSet)

		c := &types.Config{
			Kernel: currentOpsPath + "/kernel.img",
			Boot:   currentOpsPath + "/boot.img",
		}
		expected := &types.Config{
			Boot:         nightlyPath + "/boot.img",
			UefiBoot:     path.Join(nightlyPath, "bootx64.efi"),
			Kernel:       nightlyPath + "/kernel.img",
			CloudConfig:  types.ProviderConfig{},
			RunConfig:    types.RunConfig{},
			NameServers:  []string{"8.8.8.8"},
			NightlyBuild: true,
		}

		nightlyFlags.MergeToConfig(c)

		assert.Equal(t, expected, c)
	})
}
