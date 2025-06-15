package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateVolumeCommand(t *testing.T) {
	createVolumeCmd := VolumeCommands()

	createVolumeCmd.SetArgs([]string{"create", "test"})

	err := createVolumeCmd.Execute()

	assert.Nil(t, err)
}

func TestListVolumesCommand(t *testing.T) {
	listVolumesCmd := VolumeCommands()

	listVolumesCmd.SetArgs([]string{"list"})

	err := listVolumesCmd.Execute()

	assert.Nil(t, err)
}

func TestDeleteVolumeCommand(t *testing.T) {
	deleteVolumeCmd := VolumeCommands()

	deleteVolumeCmd.SetArgs([]string{"delete", "test"})

	err := deleteVolumeCmd.Execute()

	assert.Nil(t, err)
}
