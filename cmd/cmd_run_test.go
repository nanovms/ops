//go:build linux

package cmd

import (
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRunCommand(t *testing.T) {
	programPath := testutils.BuildBasicProgram()
	defer os.Remove(programPath)

	runCmd := RunCommand()

	runCmd.SetArgs([]string{programPath})

	err := runCmd.Execute()

	assert.Nil(t, err)
	removeImage(programPath)
}
