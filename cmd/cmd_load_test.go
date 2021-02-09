package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	program := buildNodejsProgram()
	defer os.Remove(program)

	loadCmd := cmd.LoadCommand()

	loadCmd.SetArgs([]string{"node_v14.2.0", "-a", program})

	err := loadCmd.Execute()

	assert.Nil(t, err)

	removeImage("node.img")
}
