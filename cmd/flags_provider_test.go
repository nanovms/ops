package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateProviderFlags(t *testing.T) {

	flagSet := pflag.NewFlagSet("test", 0)

	cmd.PersistProviderCommandFlags(flagSet)

	flagSet.Set("target-cloud", "gcp")

	providerFlags := cmd.NewProviderCommandFlags(flagSet)

	assert.Equal(t, providerFlags.TargetCloud, "gcp")
}
