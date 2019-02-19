package lepton

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type imageFilePath struct {
	Program         string `json:"program"`
	ProgramTempPath string `json:"program_temp_path"`
}

var (
	imagesOpsDir  = "images"
	imagesOpsFile = "contents.json"
)

func GetImagePathByProgramPath(path string) (imagePath string, err error) {
	var imageFilePaths []imageFilePath
	imagesOpsDirPath := filepath.Join(GetOpsHome(), imagesOpsDir)
	imagesOpsFilePath := filepath.Join(imagesOpsDirPath, imagesOpsFile)

	// Check if images Ops directory exists
	if _, err = os.Stat(imagesOpsDirPath); os.IsNotExist(err) {
		if err = os.MkdirAll(imagesOpsDirPath, 0755); err != nil {
			return
		}
	}

	// Check if file which describes contents of images data exists
	if _, err = os.Stat(imagesOpsFilePath); os.IsNotExist(err) {
		var data []tmpFilePath
		dataJSON, _ := json.Marshal(data)
		if err = ioutil.WriteFile(imagesOpsFilePath, dataJSON, 0644); err != nil {
			return
		}
	}

	// Read contents of file which describes contents of images data
	contentJSON, err := ioutil.ReadFile(imagesOpsFilePath)
	if err = json.Unmarshal(contentJSON, &imageFilePaths); err != nil {
		return
	}

	// Search for existing image paths in our file
	// If found then return
	// If found but not exist - create and return
	for _, imageFilePath := range imageFilePaths {
		if imageFilePath.Program == path {
			imagePath = imageFilePath.ProgramTempPath
			if _, err = os.Stat(imagePath); os.IsNotExist(err) {
				if err = os.MkdirAll(imagePath, 0755); err != nil {
					return
				}
			}
			return
		}
	}

	// Section where we didn't find temp path
	// First we create new directory
	imagePath, err = ioutil.TempDir(imagesOpsDirPath, "")
	if err != nil {
		return "", err
	}

	// Second we append new path to our json file
	newPath := imageFilePath{Program: path, ProgramTempPath: imagePath}
	imageFilePaths = append(imageFilePaths, newPath)
	newDataJSON, _ := json.Marshal(imageFilePaths)
	if err = ioutil.WriteFile(imagesOpsFilePath, newDataJSON, 0644); err != nil {
		return "", err
	}

	return
}

func RemoveImagePathByProgramPath(path string) error {
	var imageFilePaths []imageFilePath
	imagesOpsDirPath := filepath.Join(GetOpsHome(), imagesOpsDir)
	imagesOpsFilePath := filepath.Join(imagesOpsDirPath, imagesOpsFile)

	// Check if images Ops directory exists
	if _, err := os.Stat(imagesOpsDirPath); os.IsNotExist(err) {
		return nil
	}

	// Check if file which describes contents of images data exists
	if _, err := os.Stat(imagesOpsFilePath); os.IsNotExist(err) {
		return nil
	}

	// Read contents of file which describes contents of images data
	contentJSON, err := ioutil.ReadFile(imagesOpsFilePath)
	if err = json.Unmarshal(contentJSON, &imageFilePaths); err != nil {
		return err
	}

	// Search for existing image paths in our file
	// If found then return
	// If found but not exist - create and return
	for index, imageFilePath := range imageFilePaths {
		if imageFilePath.Program == path {
			imagePath := imageFilePath.ProgramTempPath
			if _, err = os.Stat(imagePath); os.IsNotExist(err) {
				return nil
			}
			if err = os.RemoveAll(imagePath); err != nil {
				return err
			}
			imageFilePaths = append(imageFilePaths[:index], imageFilePaths[index+1:]...)
			break
		}
	}

	newDataJSON, _ := json.Marshal(imageFilePaths)
	if err = ioutil.WriteFile(imagesOpsFilePath, newDataJSON, 0644); err != nil {
		return err
	}

	return nil
}
