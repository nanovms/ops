package crossbuild

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

var (
	// ErrMsgEnvironmentInitFailed occurs if environment initialization failed.
	ErrMsgEnvironmentInitFailed = "Failed to initialize crossbuild environment"

	// CrossBuildHomeDirPath is the crossbuild home directory.
	CrossBuildHomeDirPath = filepath.Join(lepton.GetOpsHome(), "crossbuild")

	// SupportedEnvironmentsFileName is name of the file that list supported environments.
	SupportedEnvironmentsFileName = "environments.json"

	// EnvironmentImageDirPath is the directory that keeps environment images.
	EnvironmentImageDirPath = filepath.Join(CrossBuildHomeDirPath, "images")
)

func init() {
	directories := []string{
		CrossBuildHomeDirPath,
		EnvironmentImageDirPath,
	}
	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				log.Errorf(ErrMsgEnvironmentInitFailed+": %s", err.Error())
			}
		}
	}
}

// DefaultEnvironment returns default environment.
func DefaultEnvironment() (*Environment, error) {
	environments, err := supportedEnvironments()
	if err != nil {
		return nil, err
	}
	for _, env := range environments {
		if env.IsDefault {
			return &env, nil
		}
	}
	return &environments[0], nil
}

// supportedEnvironments returns list of supported environments.
func supportedEnvironments() ([]Environment, error) {
	listFilePath := filepath.Join(CrossBuildHomeDirPath, SupportedEnvironmentsFileName)
	if _, err := os.Stat(listFilePath); os.IsNotExist(err) {
		if err = downloadEnvList(); err != nil {
			return nil, err
		}
	}
	content, err := os.ReadFile(listFilePath)
	if err != nil {
		return nil, err
	}
	var environments []Environment
	if err = json.Unmarshal(content, &environments); err != nil {
		return nil, err
	}
	return environments, nil
}

// downloadEnvList downloads list of supported environments.
func downloadEnvList() error {
	targetPath := filepath.Join(CrossBuildHomeDirPath, SupportedEnvironmentsFileName)
	downloadURL := "https://" + filepath.Join(EnvironmentDownloadBaseURL, SupportedEnvironmentsFileName)
	if err := lepton.DownloadFile(targetPath, downloadURL, 30, true); err != nil {
		return err
	}
	return nil
}
