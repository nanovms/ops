package lepton

import (
	"errors"
	"os"
)

var homeDir = ""

// HomeDir returns the home directory for the executing user.
// This uses an OS-specific method for discovering the home directory.
// An error is returned if a home directory cannot be detected.
func HomeDir() (string, error) {
	if homeDir != "" {
		return homeDir, nil
	}

	var err error = nil

	if homeDir, err = os.UserHomeDir(); err == nil {
		return homeDir, nil
	}

	homeDir, err = os.Getwd()
	if err != nil {
		return "", errors.New("home directory not detected")
	}

	return homeDir, nil
}
