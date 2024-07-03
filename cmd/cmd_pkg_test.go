package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListPkgCommand(t *testing.T) {
	listPkgCmd := PackageCommands()

	listPkgCmd.SetArgs([]string{"list"})

	err := listPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestGetPkgCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"get", "eyberg/bind:9.13.4", "--arch", "amd64"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgContentsCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"contents", "eyberg/bind:9.13.4"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgDescribeCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"describe", "eyberg/bind:9.13.4", "--arch", "amd64"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestLoad(t *testing.T) {

	getPkgCmd := PackageCommands()

	program := buildNodejsProgram()
	defer os.Remove(program)

	getPkgCmd.SetArgs([]string{"load", "eyberg/node:v14.2.0", "--arch", "amd64", "-a", program})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)

	removeImage(program)
}

func TestPkgSearch(t *testing.T) {
	searchPkgCmd := PackageCommands()

	searchPkgCmd.SetArgs([]string{"search", "mysql"})
	err := searchPkgCmd.Execute()
	assert.Nil(t, err)
}
