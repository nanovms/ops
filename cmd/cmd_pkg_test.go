package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestListPkgCommand(t *testing.T) {
	listPkgCmd := cmd.PackageCommands()

	listPkgCmd.SetArgs([]string{"list"})

	err := listPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestGetPkgCommand(t *testing.T) {
	getPkgCmd := cmd.PackageCommands()

	getPkgCmd.SetArgs([]string{"get", "node_v14.2.0"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgContentsCommand(t *testing.T) {
	getPkgCmd := cmd.PackageCommands()

	getPkgCmd.SetArgs([]string{"contents", "node_v14.2.0"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgDescribeCommand(t *testing.T) {
	getPkgCmd := cmd.PackageCommands()

	getPkgCmd.SetArgs([]string{"describe", "node_v14.2.0"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestLoad(t *testing.T) {
	getPkgCmd := cmd.PackageCommands()

	program := buildNodejsProgram()
	defer os.Remove(program)

	getPkgCmd.SetArgs([]string{"load", "node_v14.2.0", "-a", program})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)

	removeImage(program)
}
