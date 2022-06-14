package cmd_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/stretchr/testify/assert"
)

var (
	nodejsProgram = `console.log("hello world");`
)

func buildNodejsProgram() (path string) {
	program := []byte(nodejsProgram)
	randomString := testutils.String(5)
	path = "nodejs-" + randomString + ".js"

	err := ioutil.WriteFile(path, program, 0644)
	if err != nil {
		log.Panic(err)
	}

	return
}

func getImagePath(imageName string) string {
	return lepton.GetOpsHome() + "/images/" + imageName
}

func buildImage(imageName string) string {
	imageName += testutils.String(5)
	basicProgram := testutils.BuildBasicProgram()
	defer os.Remove(basicProgram)

	createImageCmd := cmd.ImageCommands()

	createImageCmd.SetArgs([]string{"create", basicProgram, "-i", imageName})

	createImageCmd.Execute()

	return imageName
}

func buildInstance(imageName string) string {
	instanceName := imageName + testutils.String(5)
	createInstanceCmd := cmd.InstanceCommands()

	createInstanceCmd.SetArgs([]string{"create", imageName, "--instance-name", instanceName})

	err := createInstanceCmd.Execute()
	if err != nil {
		log.Error(err)
	}

	return instanceName
}

func removeInstance(instanceName string) {
	createInstanceCmd := cmd.InstanceCommands()

	createInstanceCmd.SetArgs([]string{"delete", instanceName})

	createInstanceCmd.Execute()
}

func assertImageExists(t *testing.T, imageName string) {
	t.Helper()

	imagePath := getImagePath(imageName)
	_, err := os.Stat(imagePath)

	assert.False(t, os.IsNotExist(err))
}

func assertImageDoesNotExist(t *testing.T, imageName string) {
	t.Helper()

	imagePath := getImagePath(imageName)
	_, err := os.Stat(imagePath)

	assert.True(t, os.IsNotExist(err))
}

func removeImage(imageName string) {
	imagePath := getImagePath(imageName)

	os.Remove(imagePath)
}

func buildVolume(volumeName string) string {
	volumeName += testutils.String(5)
	createVolumeCmd := cmd.VolumeCommands()

	createVolumeCmd.SetArgs([]string{"create", volumeName})

	createVolumeCmd.Execute()

	return volumeName
}

func removeVolume(volumeName string) {
	createVolumeCmd := cmd.VolumeCommands()

	createVolumeCmd.SetArgs([]string{"delete", volumeName})

	createVolumeCmd.Execute()
}

func attachVolume(instanceName, volumeName string) {
	attachVolumeCmd := cmd.VolumeCommands()

	attachVolumeCmd.SetArgs([]string{"attach", instanceName, volumeName, "does not matter"})

	attachVolumeCmd.Execute()
}

func writeConfigToFile(config *types.Config, fileName string) {
	json, _ := json.MarshalIndent(config, "", "  ")

	err := ioutil.WriteFile(fileName, json, 0666)
	if err != nil {
		log.Panic(err)
	}
}
