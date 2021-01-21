package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	versionCmd := cmd.VersionCommand()

	err := versionCmd.Execute()

	assert.Nil(t, err)
}
