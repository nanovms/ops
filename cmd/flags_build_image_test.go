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

	flagSet := newBuildImageFlagSet()

	flagSet.Set("envs", "test=1234")
	flagSet.Set("target-root", "unix")
	flagSet.Set("imagename", "test-image")
	flagSet.Set("mounts", "label:path,label2:path2")
	flagSet.Set("args", "a b c d")

	buildImageFlags := cmd.NewBuildImageCommandFlags(flagSet)

	assert.Equal(t, buildImageFlags.CmdEnvs, []string{"test=1234"})
	assert.Equal(t, buildImageFlags.TargetRoot, "unix")
	assert.Equal(t, buildImageFlags.ImageName, "test-image")
	assert.Equal(t, buildImageFlags.Mounts, []string{"label:path,label2:path2"})
	assert.Equal(t, buildImageFlags.CmdArgs, []string{"a b c d"})
}

func TestBuildImageFlagsMergeToConfig(t *testing.T) {
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
		Boot:   opsPath + "/boot.img",
		Kernel: opsPath + "/kernel.img",
		Mounts: map[string]string{
			volumeName: "/files",
		},
		CloudConfig: types.ProviderConfig{
			ImageName: "test-image",
		},
		RunConfig: types.RunConfig{
			Imagename: imagesPath + "/test-image.img",
		},
		Args:       []string{"MyTestApp", "a b c d"},
		Program:    "MyTestApp",
		TargetRoot: "unix",
		Env:        map[string]string{"test": "1234"},
		NameServer: "8.8.8.8",
		VolumesDir: lepton.LocalVolumeDir,
	}

	err := buildImageFlags.MergeToConfig(c)

	assert.Nil(t, err)

	c.RunConfig.Mounts = nil

	assert.Equal(t, expected, c)
}

func newBuildImageFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistBuildImageCommandFlags(flagSet)
	return
}
