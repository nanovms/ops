package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCreateInstance(t *testing.T) {
	imageName := "test-create-instance"
	imageName = buildImage(imageName)
	defer removeImage(imageName)

	instanceName := imageName + testutils.String(5)

	createInstanceCmd := cmd.InstanceCommands()

	createInstanceCmd.SetArgs([]string{"create", instanceName, "--imagename", imageName})

	err := createInstanceCmd.Execute()

	assert.Nil(t, err)

	removeInstance(instanceName)
}

func TestListInstances(t *testing.T) {
	listInstancesCmd := cmd.InstanceCommands()

	listInstancesCmd.SetArgs([]string{"list"})

	err := listInstancesCmd.Execute()

	assert.Nil(t, err)
}

func TestDeleteInstance(t *testing.T) {
	imageName := buildImage("img-test")
	defer removeImage(imageName)
	instanceName := buildInstance(imageName)

	deleteInstanceCmd := cmd.InstanceCommands()

	deleteInstanceCmd.SetArgs([]string{"delete", instanceName})

	err := deleteInstanceCmd.Execute()

	assert.Nil(t, err)
}
