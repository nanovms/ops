package cmd_test

import (
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
			VolumesDir: lepton.LocalVolumeDir,
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
			VolumesDir: lepton.LocalVolumeDir,
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

func newConfigFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistConfigCommandFlags(flagSet)
	return
}
