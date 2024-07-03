package cmd

import (
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCmdBuild(t *testing.T) {
	programPath := testutils.BuildBasicProgram()
	defer os.Remove(programPath)

	buildCmd := BuildCommand()

	buildCmd.SetArgs([]string{programPath})

	err := buildCmd.Execute()

	assert.Nil(t, err)

	imageName := programPath

	assertImageExists(t, imageName)
	removeImage(imageName)
}
