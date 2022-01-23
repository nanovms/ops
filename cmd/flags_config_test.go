package cmd_test

import (
	"errors"
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestConfigFlags(t *testing.T) {

	flagSet := newConfigFlagSet()

	flagSet.Set("config", "test.json")

	buildImageFlags := cmd.NewConfigCommandFlags(flagSet)

	assert.Equal(t, buildImageFlags.Config, "test.json")
}

func TestConfigFlagsMergeToConfig(t *testing.T) {

	t.Run("should merge file configuration", func(t *testing.T) {
		configFileName := "test-" + testutils.String(5) + ".json"
		expected := &types.Config{
			CloudConfig: types.ProviderConfig{
				ProjectID:  "projectid",
				BucketName: "thebucketname",
			},
			RunConfig: types.RunConfig{
				Memory:    "2G",
				IPAddress: "192.168.1.5",
				NetMask:   "255.255.255.0",
				Gateway:   "192.168.1.254",
			},
			VolumesDir:                lepton.LocalVolumeDir,
			LocalFilesParentDirectory: ".",
		}

		writeConfigToFile(expected, configFileName)
		defer os.Remove(configFileName)

		flagSet := newConfigFlagSet()
		flagSet.Set("config", configFileName)
		configFlags := cmd.NewConfigCommandFlags(flagSet)

		actual := &types.Config{}

		err := configFlags.MergeToConfig(actual)

		assert.Nil(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("should ignore ip/gateway/netmask if they are not valid ip addresses", func(t *testing.T) {
		configFileName := "test-" + testutils.String(5) + ".json"
		expected := &types.Config{
			CloudConfig: types.ProviderConfig{
				ProjectID:  "projectid",
				BucketName: "thebucketname",
			},
			RunConfig: types.RunConfig{
				Memory: "2G",
			},
			VolumesDir:                lepton.LocalVolumeDir,
			LocalFilesParentDirectory: ".",
		}
		config := &types.Config{
			CloudConfig: types.ProviderConfig{
				ProjectID:  "projectid",
				BucketName: "thebucketname",
			},
			RunConfig: types.RunConfig{
				Memory:    "2G",
				IPAddress: "tomato",
				NetMask:   "potato",
				Gateway:   "cheese",
			},
			VolumesDir: lepton.LocalVolumeDir,
		}

		writeConfigToFile(config, configFileName)
		defer os.Remove(configFileName)

		flagSet := newConfigFlagSet()
		flagSet.Set("config", configFileName)
		configFlags := cmd.NewConfigCommandFlags(flagSet)

		actual := &types.Config{}

		err := configFlags.MergeToConfig(actual)

		assert.Nil(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestConvertJsonToConfig(t *testing.T) {
	t.Run("should return error if property does not exist in config struct", func(t *testing.T) {
		json := []byte(`{"A":1}`)
		config := &types.Config{}

		err := cmd.ConvertJSONToConfig(json, config)

		assert.Error(t, err)
		assert.Equal(t, errors.Unwrap(err), errors.Unwrap(cmd.ErrInvalidFileConfig(errors.New("json: unknown field \"A\""))))
	})

	t.Run("should return error if nested property does not exist in config struct", func(t *testing.T) {
		json := []byte(`{"RunConfig":{"A":1}}`)
		config := &types.Config{}

		err := cmd.ConvertJSONToConfig(json, config)

		assert.Error(t, err)
		assert.Equal(t, errors.Unwrap(err), errors.Unwrap(cmd.ErrInvalidFileConfig(errors.New("json: unknown field \"A\""))))
	})

	t.Run("should return error if some properties do not exist in config struct", func(t *testing.T) {
		json := []byte(`{"RunConfig":{"Accel":false,"Test2":"b","Test3":"c"}}`)
		config := &types.Config{}

		err := cmd.ConvertJSONToConfig(json, config)

		assert.Error(t, err)
		assert.Equal(t, errors.Unwrap(err), errors.Unwrap(cmd.ErrInvalidFileConfig(errors.New("json: unknown field \"Test2\""))))
	})

	t.Run("should return error if properties in first level do not exist in config struct", func(t *testing.T) {
		json := []byte(`{"RunConfig":{"Accel":false},"test":"b"}`)
		config := &types.Config{}

		err := cmd.ConvertJSONToConfig(json, config)

		assert.Error(t, err)
		assert.Equal(t, errors.Unwrap(err), errors.Unwrap(cmd.ErrInvalidFileConfig(errors.New("json: unknown field \"test\""))))
	})

	t.Run("should convert config successfully if all properties are known", func(t *testing.T) {
		json := []byte(`{"RunConfig":{"Accel":false}}`)
		config := &types.Config{}

		err := cmd.ConvertJSONToConfig(json, config)

		assert.Nil(t, err)
	})
}

func newConfigFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistConfigCommandFlags(flagSet)
	return
}
