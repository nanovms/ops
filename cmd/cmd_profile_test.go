package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileCommand(t *testing.T) {
	profileCommand := ProfileCommand()

	err := profileCommand.Execute()

	assert.Nil(t, err)
}
