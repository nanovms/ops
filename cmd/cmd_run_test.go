package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRunCommand(t *testing.T) {

	stubUpdate()

	programPath := testutils.BuildBasicProgram()
	defer os.Remove(programPath)

	runCmd := cmd.RunCommand()

	runCmd.SetArgs([]string{programPath})

	err := runCmd.Execute()

	assert.Nil(t, err)
	removeImage(programPath)
}
