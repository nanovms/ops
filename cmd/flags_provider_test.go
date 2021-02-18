package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateProviderFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistProviderCommandFlags(flagSet)

	flagSet.Set("target-cloud", "gcp")
	flagSet.Set("zone", "us-west")
	flagSet.Set("projectid", "prod-xpto")

	providerFlags := cmd.NewProviderCommandFlags(flagSet)

	assert.Equal(t, providerFlags.TargetCloud, "gcp")
	assert.Equal(t, providerFlags.Zone, "us-west")
	assert.Equal(t, providerFlags.Project, "prod-xpto")
}

func TestMergeProviderFlags(t *testing.T) {

	t.Run("target cloud azure should set klib cloud_init", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistProviderCommandFlags(flagSet)

		flagSet.Set("target-cloud", "azure")

		providerFlags := cmd.NewProviderCommandFlags(flagSet)

		actual := &config.Config{}
		expected := &config.Config{
			CloudConfig: config.ProviderConfig{
				Platform: "azure",
			},
			RunConfig: config.RunConfig{
				Klibs: []string{"cloud_init"},
			},
		}

		providerFlags.MergeToConfig(actual)

		assert.Equal(t, expected, actual)
	})

	t.Run("should update config with flags values", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		cmd.PersistProviderCommandFlags(flagSet)

		flagSet.Set("target-cloud", "gcp")
		flagSet.Set("zone", "us-west")
		flagSet.Set("projectid", "prod-xpto")

		providerFlags := cmd.NewProviderCommandFlags(flagSet)

		actual := &config.Config{}
		expected := &config.Config{
			CloudConfig: config.ProviderConfig{
				Platform:  "gcp",
				Zone:      "us-west",
				ProjectID: "prod-xpto",
			},
			RunConfig: config.RunConfig{},
		}

		providerFlags.MergeToConfig(actual)

		assert.Equal(t, expected, actual)
	})
}
