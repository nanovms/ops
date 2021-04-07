package wsl

import (
	"os/exec"
	"strings"
)

func IsWSL() bool {
	_, err := exec.LookPath("wslpath")

	return err == nil
}

func ConvertPathFromWSLtoWindows(path string) (windowsPath string, err error) {
	cmd := exec.Command("wslpath", "-w", path)

	bytesWindowsPath, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	windowsPath = strings.Trim(string(bytesWindowsPath), "\n")

	return
}
