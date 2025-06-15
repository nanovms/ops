package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"
	"github.com/stretchr/testify/assert"
)

func TestDetachVolumeCommand(t *testing.T) {
	imageName := buildWaitImage("test")
	defer removeImage(imageName)

	configFileName := "test-" + testutils.String(5) + ".json"
	expected := &types.Config{
		RunConfig: types.RunConfig{
			QMP: true,
		},
	}

	writeConfigToFile(expected, configFileName)
	defer os.Remove(configFileName)

	instanceName := buildInstanceWithConfig(imageName, configFileName)
	defer removeInstance(instanceName)

	volumeName := buildVolume("vol-test")
	defer removeVolume(volumeName)

	attachVolumeCmd := VolumeCommands()

	attachVolumeCmd.SetArgs([]string{"attach", instanceName, volumeName, "does not matter"})

	// FIXME: tests prob. should not be spawning
	time.Sleep(1 * time.Second)

	err := attachVolumeCmd.Execute()

	assert.Nil(t, err)

	detachVolumeCmd := VolumeCommands()

	detachVolumeCmd.SetArgs([]string{"detach", instanceName, volumeName})

	err = detachVolumeCmd.Execute()

	assert.Nil(t, err)
}
