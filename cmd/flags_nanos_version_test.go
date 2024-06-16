package cmd

import (
	"os"
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

// stubs a 'release'
func stubRelease(versionPath string) error {
	err := os.MkdirAll(versionPath, 0750)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(versionPath+"/kernel.img", os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

func TestVersionFlagsMergeToConfig(t *testing.T) {

	versionPath := path.Join(lepton.GetOpsHome(), "0.1.37")

	a := archPath()

	if a != "" {
		versionPath = versionPath + "-" + a
	}

	err := stubRelease(versionPath)
	if err != nil {
		t.Fatal(err)
	}

	currentOpsPath := path.Join(lepton.GetOpsHome(), lepton.LocalReleaseVersion)

	t.Run("if nano-version flag is enabled should set boot and kernel paths", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistNanosVersionCommandFlags(flagSet)

		flagSet.Set("nanos-version", "0.1.37")

		versionFlags := NewNanosVersionCommandFlags(flagSet)

		boot := path.Join(versionPath, "boot.img")
		if a != "" {
			boot = ""
		}

		c := &types.Config{}
		expected := &types.Config{
			Boot:   boot,
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

		cBoot := currentOpsPath + "/boot.img"
		vBoot := versionPath + "/boot.img"

		if a != "" {
			cBoot = ""
			vBoot = ""
		}

		c := &types.Config{
			Kernel: currentOpsPath + "/kernel.img",
			Boot:   cBoot,
		}
		expected := &types.Config{
			Boot:   vBoot,
			Kernel: versionPath + "/kernel.img",
		}

		versionFlags.MergeToConfig(c)

		assert.Equal(t, expected.Boot, c.Boot)
		assert.Equal(t, expected.Kernel, c.Kernel)

	})
}
