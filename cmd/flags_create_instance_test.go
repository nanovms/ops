package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateCreateInstanceFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistCreateInstanceFlags(flagSet)

	flagSet.Set("domainname", "test.nanovms.com")
	flagSet.Set("flavor", "t2")
	flagSet.Set("port", "80,9000-9040")
	flagSet.Set("udp", "50-80,8000")

	createInstanceFlags := cmd.NewCreateInstanceCommandFlags(flagSet)

	assert.Equal(t, createInstanceFlags.DomainName, "test.nanovms.com")
	assert.Equal(t, createInstanceFlags.Flavor, "t2")
	assert.Equal(t, createInstanceFlags.Ports, []string{"80", "9000-9040"})
	assert.Equal(t, createInstanceFlags.UDPPorts, []string{"50-80", "8000"})

}

func TestCreateInstanceFlagsMergeToConfig(t *testing.T) {
	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistCreateInstanceFlags(flagSet)

	flagSet.Set("domainname", "test.nanovms.com")
	flagSet.Set("flavor", "t2")
	flagSet.Set("port", "80,9000-9040")
	flagSet.Set("udp", "50-80,8000")

	createInstanceFlags := cmd.NewCreateInstanceCommandFlags(flagSet)

	expected := &config.Config{
		CloudConfig: config.ProviderConfig{
			Flavor: "t2",
		},
		RunConfig: config.RunConfig{
			DomainName: "test.nanovms.com",
			Ports:      []string{"30", "80", "9000-9040"},
			UDPPorts:   []string{"90", "50-80", "8000"},
		},
	}

	actual := &config.Config{
		RunConfig: config.RunConfig{
			Ports:    []string{"30"},
			UDPPorts: []string{"90"},
		},
	}

	err := createInstanceFlags.MergeToConfig(actual)

	assert.Nil(t, err)

	assert.Equal(t, expected, actual)
}
