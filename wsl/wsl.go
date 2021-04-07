package wsl

import (
	"os/exec"
	"strings"
)

// IsWSL checks whether current operating system is a WSL distro
func IsWSL() bool {
	_, err := exec.LookPath("wslpath")

	return err == nil
}

// ConvertPathFromWSLtoWindows maps WSL paths to Windows paths
func ConvertPathFromWSLtoWindows(path string) (windowsPath string, err error) {
	cmd := exec.Command("wslpath", "-w", path)

	bytesWindowsPath, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	windowsPath = strings.Trim(string(bytesWindowsPath), "\n")

	return
}
