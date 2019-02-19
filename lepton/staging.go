package lepton

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type tmpFilePath struct {
	Program         string `json:"program"`
	ProgramTempPath string `json:"program_temp_path"`
}

var (
	tempOpsDir  = "staging"
	tempOpsFile = "contents.json"
)

func GetTMPPathByProgramPath(path string) (tempPath string, err error) {
	var tmpFilePaths []tmpFilePath
	tempOpsDirPath := filepath.Join(GetOpsHome(), tempOpsDir)
	tempOpsFilePath := filepath.Join(tempOpsDirPath, tempOpsFile)

	// Check if temp Ops directory exists
	if _, err = os.Stat(tempOpsDirPath); os.IsNotExist(err) {
		if err = os.MkdirAll(tempOpsDirPath, 0755); err != nil {
			return
		}
	}

	// Check if file which describes contents of tmp data exists
	if _, err = os.Stat(tempOpsFilePath); os.IsNotExist(err) {
		var data []tmpFilePath
		dataJSON, _ := json.Marshal(data)
		if err = ioutil.WriteFile(tempOpsFilePath, dataJSON, 0644); err != nil {
			return
		}
	}

	// Read contents of file which describes contents of tmp data
	contentJSON, err := ioutil.ReadFile(tempOpsFilePath)
	if err = json.Unmarshal(contentJSON, &tmpFilePaths); err != nil {
		return
	}

	// Search for existing temp paths in our file
	// If found then return
	// If found but not exist - create and return
	for _, tmpFilePath := range tmpFilePaths {
		if tmpFilePath.Program == path {
			tempPath = tmpFilePath.ProgramTempPath
			if _, err = os.Stat(tempPath); os.IsNotExist(err) {
				if err = os.MkdirAll(tempPath, 0755); err != nil {
					return
				}
			}
			return
		}
	}

	// Section where we didn't find temp path
	// First we create new directory
	tempPath, err = ioutil.TempDir(tempOpsDirPath, "")
	if err != nil {
		return "", err
	}

	// Second we append new path to our json file
	newPath := tmpFilePath{Program: path, ProgramTempPath: tempPath}
	tmpFilePaths = append(tmpFilePaths, newPath)
	newDataJSON, _ := json.Marshal(tmpFilePaths)
	if err = ioutil.WriteFile(tempOpsFilePath, newDataJSON, 0644); err != nil {
		return "", err
	}

	return
}

func RemoveTMPPathByProgramPath(path string) error {
	var tmpFilePaths []tmpFilePath
	tempOpsDirPath := filepath.Join(GetOpsHome(), tempOpsDir)
	tempOpsFilePath := filepath.Join(tempOpsDirPath, tempOpsFile)

	// Check if temp Ops directory exists
	if _, err := os.Stat(tempOpsDirPath); os.IsNotExist(err) {
		return nil
	}

	// Check if file which describes contents of tmp data exists
	if _, err := os.Stat(tempOpsFilePath); os.IsNotExist(err) {
		return nil
	}

	// Read contents of file which describes contents of tmp data
	contentJSON, err := ioutil.ReadFile(tempOpsFilePath)
	if err = json.Unmarshal(contentJSON, &tmpFilePaths); err != nil {
		return err
	}

	// Search for existing temp paths in our file
	// If found then return
	// If found but not exist - create and return
	for index, tmpFilePath := range tmpFilePaths {
		if tmpFilePath.Program == path {
			tempPath := tmpFilePath.ProgramTempPath
			if _, err = os.Stat(tempPath); os.IsNotExist(err) {
				return nil
			}
			if err = os.RemoveAll(tempPath); err != nil {
				return err
			}
			tmpFilePaths = append(tmpFilePaths[:index], tmpFilePaths[index+1:]...)
			break
		}
	}

	newDataJSON, _ := json.Marshal(tmpFilePaths)
	if err = ioutil.WriteFile(tempOpsFilePath, newDataJSON, 0644); err != nil {
		return err
	}

	return nil
}
