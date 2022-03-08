package cmd

import (
	"path"
	"testing"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateVersionFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	PersistNanosVersionCommandFlags(flagSet)

	flagSet.Set("nanos-version", "0.1.37")

	versionFlags := NewNanosVersionCommandFlags(flagSet)

	assert.Equal(t, versionFlags.NanosVersion, "0.1.37")
}

func TestVersionFlagsMergeToConfig(t *testing.T) {

	versionPath := path.Join(lepton.GetOpsHome(), "0.1.37")
	currentOpsPath := path.Join(lepton.GetOpsHome(), lepton.LocalReleaseVersion)

	t.Run("if nano-version flag is enabled should set boot and kernel paths", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistNanosVersionCommandFlags(flagSet)

		flagSet.Set("nanos-version", "0.1.37")

		versionFlags := NewNanosVersionCommandFlags(flagSet)

		c := &types.Config{}
		expected := &types.Config{
			Boot:   path.Join(versionPath, "boot.img"),
			Kernel: path.Join(versionPath, "kernel.img"),
		}

		versionFlags.MergeToConfig(c)

		assert.Equal(t, expected.Boot, c.Boot)
		assert.Equal(t, expected.Kernel, c.Kernel)

	})

	t.Run("if nanos-version flag is enabled should override current configuration boot and kernel paths with version paths", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistNanosVersionCommandFlags(flagSet)

		flagSet.Set("nanos-version", "0.1.37")

		versionFlags := NewNanosVersionCommandFlags(flagSet)

		c := &types.Config{
			Kernel: currentOpsPath + "/kernel.img",
			Boot:   currentOpsPath + "/boot.img",
		}
		expected := &types.Config{
			Boot:   versionPath + "/boot.img",
			Kernel: versionPath + "/kernel.img",
		}

		versionFlags.MergeToConfig(c)

		assert.Equal(t, expected.Boot, c.Boot)
		assert.Equal(t, expected.Kernel, c.Kernel)

	})
}
