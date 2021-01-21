package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestCmdBuild(t *testing.T) {
	programPath := buildBasicProgram()
	defer os.Remove(programPath)

	buildCmd := cmd.BuildCommand()

	buildCmd.SetArgs([]string{programPath})

	err := buildCmd.Execute()

	assert.Nil(t, err)

	imageName := programPath

	assertImageExists(t, imageName)
	removeImage(imageName)
}
