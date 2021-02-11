package cmd_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	"github.com/stretchr/testify/assert"
)

var (
	basicProgram = `package main

	import(
		"fmt"
	)

	func main(){
		fmt.Println("hello world")
	}
	`
	nodejsProgram = `console.log("hello world");`
)

func buildBasicProgram() (binaryPath string) {
	program := []byte(basicProgram)
	randomString := String(5)
	binaryPath = "./basic" + randomString
	sourcePath := fmt.Sprintf("basic%s.go", randomString)

	err := ioutil.WriteFile(sourcePath, program, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer os.Remove(sourcePath)

	cmd := exec.Command("go", "build", sourcePath)
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	return
}

func buildNodejsProgram() (path string) {
	program := []byte(nodejsProgram)
	randomString := String(5)
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
	imageName += String(5)
	basicProgram := buildBasicProgram()
	defer os.Remove(basicProgram)

	createImageCmd := cmd.ImageCommands()

	createImageCmd.SetArgs([]string{"create", "-a", basicProgram, "-i", imageName})

	createImageCmd.Execute()

	return imageName
}

func buildInstance(imageName string) string {
	instanceName := imageName + String(5)
	createInstanceCmd := cmd.InstanceCommands()

	createInstanceCmd.SetArgs([]string{"create", instanceName, "--imagename", imageName})

	err := createInstanceCmd.Execute()
	if err != nil {
		fmt.Println(err)
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

	imagePath := getImagePath(imageName + ".img")
	_, err := os.Stat(imagePath)

	assert.False(t, os.IsNotExist(err))
}

func assertImageDoesNotExist(t *testing.T, imageName string) {
	t.Helper()

	imagePath := getImagePath(imageName + ".img")
	_, err := os.Stat(imagePath)

	assert.True(t, os.IsNotExist(err))
}

func removeImage(imageName string) {
	imagePath := getImagePath(imageName + ".img")

	os.Remove(imagePath)
}

func buildVolume(volumeName string) string {
	volumeName += String(5)
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

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func writeConfigToFile(config *lepton.Config, fileName string) {
	json, _ := json.MarshalIndent(config, "", "  ")

	err := ioutil.WriteFile(fileName, json, 0666)
	if err != nil {
		panic(err)
	}
}
