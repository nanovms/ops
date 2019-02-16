package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	api "github.com/nanovms/ops/lepton"
)

type TMPFilePath struct {
	Program         string `json:"program"`
	ProgramTempPath string `json:"program_temp_path"`
}

func GetTMPPathByProgramPath(path string) (tempPath string, err error) {
	var tmpFilePaths []TMPFilePath
	tempDir := os.TempDir()
	tempOpsDir := filepath.Join(tempDir, api.OpsDir)
	tempOpsFile := filepath.Join(tempOpsDir, api.TMPContentsFile)

	// Check if temp Ops directory exists
	if _, err = os.Stat(tempOpsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(tempOpsDir, 0755); err != nil {
			return
		}
	}

	// Check if file which describes contents of tmp data exists
	if _, err = os.Stat(tempOpsFile); os.IsNotExist(err) {
		data := []TMPFilePath{}
		dataJSON, _ := json.Marshal(data)
		if err = ioutil.WriteFile(tempOpsFile, dataJSON, 0644); err != nil {
			return
		}
	}

	// Read contents of file which describes contents of tmp data
	contentJSON, err := ioutil.ReadFile(tempOpsFile)
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
	tempPath, err = ioutil.TempDir(tempOpsDir, api.TMPStagingDirectoryPrefix)
	if err != nil {
		return "", err
	}

	// Second we append new path to our json file
	newPath := TMPFilePath{Program: path, ProgramTempPath: tempPath}
	tmpFilePaths = append(tmpFilePaths, newPath)
	newDataJSON, _ := json.Marshal(tmpFilePaths)
	if err = ioutil.WriteFile(tempOpsFile, newDataJSON, 0644); err != nil {
		return "", err
	}

	return
}
