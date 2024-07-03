package cmd

import (
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	basicProgram := testutils.BuildBasicProgram()
	defer os.Remove(basicProgram)

	createImageCmd := ImageCommands()

	createImageCmd.SetArgs([]string{"create", basicProgram})

	err := createImageCmd.Execute()

	assert.Nil(t, err)

	imageName := basicProgram
	assertImageExists(t, imageName)
	removeImage(imageName)
}

func TestListImages(t *testing.T) {
	listImagesCmd := ImageCommands()

	listImagesCmd.SetArgs([]string{"list"})

	err := listImagesCmd.Execute()

	assert.Nil(t, err)
}

func TestDeleteImage(t *testing.T) {
	imagePath := buildImage("test-img")

	deleteImageCmd := ImageCommands()

	deleteImageCmd.SetArgs([]string{"delete", imagePath, "--assume-yes"})

	err := deleteImageCmd.Execute()

	assert.Nil(t, err)
	assertImageDoesNotExist(t, imagePath)
}
