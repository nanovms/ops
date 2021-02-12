package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	basicProgram := buildBasicProgram()
	defer os.Remove(basicProgram)

	createImageCmd := cmd.ImageCommands()

	createImageCmd.SetArgs([]string{"create", "-a", basicProgram})

	err := createImageCmd.Execute()

	assert.Nil(t, err)

	imageName := basicProgram
	assertImageExists(t, imageName)
	removeImage(imageName)
}

func TestListImages(t *testing.T) {
	listImagesCmd := cmd.ImageCommands()

	listImagesCmd.SetArgs([]string{"list"})

	err := listImagesCmd.Execute()

	assert.Nil(t, err)
}

func TestDeleteImage(t *testing.T) {
	imagePath := buildImage("test-img")

	deleteImageCmd := cmd.ImageCommands()

	deleteImageCmd.SetArgs([]string{"delete", imagePath + ".img", "--assume-yes"})

	err := deleteImageCmd.Execute()

	assert.Nil(t, err)
	assertImageDoesNotExist(t, imagePath)
}
