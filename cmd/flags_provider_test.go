package cmd

import (
	"testing"

	"github.com/nanovms/ops/types"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateProviderFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	PersistProviderCommandFlags(flagSet)

	flagSet.Set("target-cloud", "gcp")
	flagSet.Set("zone", "us-west")
	flagSet.Set("projectid", "prod-xpto")

	providerFlags := NewProviderCommandFlags(flagSet)

	assert.Equal(t, providerFlags.TargetCloud, "gcp")
	assert.Equal(t, providerFlags.Zone, "us-west")
	assert.Equal(t, providerFlags.Project, "prod-xpto")
}

func TestMergeProviderFlags(t *testing.T) {

	t.Run("target cloud azure should set klib cloud_init", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistProviderCommandFlags(flagSet)

		flagSet.Set("target-cloud", "azure")

		providerFlags := NewProviderCommandFlags(flagSet)

		actual := &types.Config{}
		expected := &types.Config{
			CloudConfig: types.ProviderConfig{
				Platform: "azure",
			},
			Klibs: []string{"cloud_init", "tls"},
		}

		providerFlags.MergeToConfig(actual)

		assert.Equal(t, expected, actual)
	})

	t.Run("should update config with flags values", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", 0)

		PersistProviderCommandFlags(flagSet)

		flagSet.Set("target-cloud", "gcp")
		flagSet.Set("zone", "us-west")
		flagSet.Set("projectid", "prod-xpto")

		providerFlags := NewProviderCommandFlags(flagSet)

		actual := &types.Config{}
		expected := &types.Config{
			CloudConfig: types.ProviderConfig{
				Platform:  "gcp",
				Zone:      "us-west",
				ProjectID: "prod-xpto",
			},
			RunConfig: types.RunConfig{},
		}

		providerFlags.MergeToConfig(actual)

		assert.Equal(t, expected, actual)
	})
}
