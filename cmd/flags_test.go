package cmd

import (
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/lepton"
	"github.com/stretchr/testify/assert"
)

func TestMergeMultipleFlags(t *testing.T) {

	buildImageFlagSet := newBuildImageFlagSet()
	buildImageFlagSet.Set("imagename", "build-image")
	buildImageFlagSet.Set("args", "test")
	buildImageFlags := NewBuildImageCommandFlags(buildImageFlagSet)

	configFileName := "test-" + testutils.String(5) + ".json"

	configFile := &types.Config{
		RunConfig: types.RunConfig{
			ImageName: "config-image-name",
		},
	}

	writeConfigToFile(configFile, configFileName)
	defer os.Remove(configFileName)
	configFlagSet := newConfigFlagSet()
	configFlagSet.Set("config", configFileName)
	configFlags := NewConfigCommandFlags(configFlagSet)

	imagesPath := lepton.GetOpsHome() + "/images"

	t.Run("if config flags are placed before the build image flags imagename overrides the value", func(t *testing.T) {
		container := NewMergeConfigContainer(configFlags, buildImageFlags)

		config := &types.Config{}

		err := container.Merge(config)

		assert.Nil(t, err)
		assert.Equal(t, config.RunConfig.ImageName, imagesPath+"/build-image")
	})

	t.Run("if build image flags are placed before the config image flags imagename overrides the value", func(t *testing.T) {
		container := NewMergeConfigContainer(buildImageFlags, configFlags)

		config := &types.Config{}

		err := container.Merge(config)

		assert.Nil(t, err)

		assert.Equal(t, config.RunConfig.ImageName, "config-image-name")
	})
}
