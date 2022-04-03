package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateBuildImageFlags(t *testing.T) {

	t.Run("should map flags to struct", func(t *testing.T) {
		flagSet := newBuildImageFlagSet()

		flagSet.Set("envs", "test=1234")
		flagSet.Set("target-root", "unix")
		flagSet.Set("imagename", "test-image")
		flagSet.Set("mounts", "label:path,label2:path2")
		flagSet.Set("args", "a b c d")
		flagSet.Set("ip-address", "192.168.0.1")
		flagSet.Set("ipv6-address", "FE80::46F:65FF:FE9C:4861")
		flagSet.Set("gateway", "192.168.1.254")
		flagSet.Set("netmask", "255.255.0.0")

		buildImageFlags := cmd.NewBuildImageCommandFlags(flagSet)

		assert.Equal(t, buildImageFlags.CmdEnvs, []string{"test=1234"})
		assert.Equal(t, buildImageFlags.TargetRoot, "unix")
		assert.Equal(t, buildImageFlags.ImageName, "test-image")
		assert.Equal(t, buildImageFlags.Mounts, []string{"label:path,label2:path2"})
		assert.Equal(t, buildImageFlags.CmdArgs, []string{"a b c d"})
		assert.Equal(t, buildImageFlags.IPAddress, "192.168.0.1")
		assert.Equal(t, buildImageFlags.IPv6Address, "FE80::46F:65FF:FE9C:4861")
		assert.Equal(t, buildImageFlags.Gateway, "192.168.1.254")
		assert.Equal(t, buildImageFlags.Netmask, "255.255.0.0")
	})
}

func TestBuildImageFlagsMergeToConfig(t *testing.T) {

	t.Run("should merge flags into configuration", func(t *testing.T) {
		volumeName := buildVolume("vol")
		defer removeVolume(volumeName)

		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistBuildImageCommandFlags(flagSet)

		flagSet.Set("args", "a b c d")
		flagSet.Set("envs", "test=1234")
		flagSet.Set("target-root", "unix")
		flagSet.Set("imagename", "test-image")
		flagSet.Set("mounts", volumeName+":/files")

		buildImageFlags := cmd.NewBuildImageCommandFlags(flagSet)

		opsPath := lepton.GetOpsHome() + "/" + lepton.LocalReleaseVersion
		imagesPath := lepton.GetOpsHome() + "/images"

		c := &types.Config{
			VolumesDir: lepton.LocalVolumeDir,
			Program:    "MyTestApp",
		}
		expected := &types.Config{
			Boot:     opsPath + "/boot.img",
			UefiBoot: opsPath + "/bootx64.efi",
			Kernel:   opsPath + "/kernel.img",
			Mounts: map[string]string{
				volumeName: "/files",
			},
			CloudConfig: types.ProviderConfig{
				ImageName: "test-image",
			},
			RunConfig: types.RunConfig{
				Imagename: imagesPath + "/test-image.img",
				NetMask:   "255.255.255.0",
			},
			Args:                []string{"MyTestApp", "a b c d"},
			Program:             "MyTestApp",
			TargetRoot:          "unix",
			Env:                 map[string]string{"test": "1234"},
			NameServers:         []string{"8.8.8.8"},
			VolumesDir:          lepton.LocalVolumeDir,
			ManifestPassthrough: map[string]interface{}{},
		}

		err := buildImageFlags.MergeToConfig(c)

		assert.Nil(t, err)

		c.RunConfig.Mounts = nil

		assert.Equal(t, expected, c)
	})

	t.Run("should clear ip/gateway/netmask if they are not valid ip addresses", func(t *testing.T) {
		flagSet := newBuildImageFlagSet()

		runLocalInstanceFlags := cmd.NewBuildImageCommandFlags(flagSet)
		runLocalInstanceFlags.IPAddress = "tomato"
		runLocalInstanceFlags.Gateway = "potato"
		runLocalInstanceFlags.Netmask = "cheese"

		c := &types.Config{}

		err := runLocalInstanceFlags.MergeToConfig(c)

		assert.Nil(t, err, nil)

		opsPath := lepton.GetOpsHome() + "/" + lepton.LocalReleaseVersion

		expected := &types.Config{
			Boot:                opsPath + "/boot.img",
			UefiBoot:            opsPath + "/bootx64.efi",
			Kernel:              opsPath + "/kernel.img",
			Mounts:              map[string]string{},
			CloudConfig:         types.ProviderConfig{},
			RunConfig:           types.RunConfig{},
			Args:                []string{},
			NameServers:         []string{"8.8.8.8"},
			ManifestPassthrough: map[string]interface{}{},
		}

		assert.Equal(t, expected, c)
	})

}

func newBuildImageFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistBuildImageCommandFlags(flagSet)
	return
}
