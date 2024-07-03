package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	versionCmd := VersionCommand()

	err := versionCmd.Execute()

	assert.Nil(t, err)
}
