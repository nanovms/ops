package crossbuild

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/nanovms/ops/lepton"
)

// Environments is collection of supported environments.
type Environments []Environment

// Find returns environment identified by given id, if any.
func (em Environments) Find(id string) *Environment {
	for _, env := range em {
		if env.ID == id {
			return &env
		}
	}
	return nil
}

// Default returns default environment from one of supported environments.
func (em Environments) Default() *Environment {
	for _, env := range em {
		if env.IsDefault {
			return &env
		}
	}
	return &em[0]
}

// SupportedEnvironments returns list of supported environments.
func SupportedEnvironments() (Environments, error) {
	listFilePath := filepath.Join(CrossBuildHomeDirPath, SupportedEnvironmentsFileName)
	if _, err := os.Stat(listFilePath); os.IsNotExist(err) {
		if err = downloadManifest(); err != nil {
			return nil, err
		}
	}

	content, err := ioutil.ReadFile(listFilePath)
	if err != nil {
		return nil, err
	}

	var environments Environments
	if err = json.Unmarshal(content, &environments); err != nil {
		return nil, err
	}
	return environments, nil
}

// Download manifest.
func downloadManifest() error {
	targetPath := filepath.Join(CrossBuildHomeDirPath, SupportedEnvironmentsFileName)
	downloadURL := "https://" + path.Join(EnvironmentDownloadBaseURL, SupportedEnvironmentsFileName)
	fmt.Println("Download list of supported environments")
	if err := lepton.DownloadFile(targetPath, downloadURL, 600, true); err != nil {
		return err
	}
	return nil
}
