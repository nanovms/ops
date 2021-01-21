package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestProfileCommand(t *testing.T) {
	profileCommand := cmd.ProfileCommand()

	err := profileCommand.Execute()

	assert.Nil(t, err)
}
