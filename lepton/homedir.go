package lepton

import (
	"errors"
	"os"
)

// HomeDir returns the home directory for the executing user.
// This uses an OS-specific method for discovering the home directory.
// An error is returned if a home directory cannot be detected.
func HomeDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil {
		return home, nil
	}

	path, err := os.Getwd()
	if err != nil {
		return "", errors.New("home directory not detected")
	}

	return path, nil
}
