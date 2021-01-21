package cmd_test

import (
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestCreateVolumeCommand(t *testing.T) {
	createVolumeCmd := cmd.VolumeCommands()

	createVolumeCmd.SetArgs([]string{"create", "test"})

	err := createVolumeCmd.Execute()

	assert.Nil(t, err)
}

func TestListVolumesCommand(t *testing.T) {
	listVolumesCmd := cmd.VolumeCommands()

	listVolumesCmd.SetArgs([]string{"list"})

	err := listVolumesCmd.Execute()

	assert.Nil(t, err)
}

func TestDeleteVolumeCommand(t *testing.T) {
	deleteVolumeCmd := cmd.VolumeCommands()

	deleteVolumeCmd.SetArgs([]string{"delete", "test"})

	err := deleteVolumeCmd.Execute()

	assert.Nil(t, err)
}

func TestAttachVolumeCommand(t *testing.T) {
	imageName := buildImage("test")
	defer removeImage(imageName)
	instanceName := buildInstance(imageName)
	defer removeInstance(instanceName)
	volumeName := buildVolume("vol-test")
	defer removeVolume(volumeName)

	attachVolumeCmd := cmd.VolumeCommands()

	attachVolumeCmd.SetArgs([]string{"attach", instanceName, volumeName, "does not matter"})

	err := attachVolumeCmd.Execute()

	assert.Nil(t, err)
}

func TestDetachVolumeCommand(t *testing.T) {
	imageName := buildImage("test")
	defer removeImage(imageName)
	instanceName := buildInstance(imageName)
	defer removeInstance(instanceName)
	volumeName := buildVolume("vol-test")
	defer removeVolume(volumeName)

	detachVolumeCmd := cmd.VolumeCommands()

	detachVolumeCmd.SetArgs([]string{"detach", instanceName, volumeName})

	err := detachVolumeCmd.Execute()

	assert.Nil(t, err)
}
