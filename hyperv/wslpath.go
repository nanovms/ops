package hyperv

import (
	"os/exec"
	"strings"
)

func convertPathFromWSLtoWindows(path string) (windowsPath string, err error) {
	_, err = exec.LookPath("wslpath")
	if err != nil {
		return
	}

	cmd := exec.Command("wslpath", "-w", path)

	bytesWindowsPath, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	windowsPath = strings.Trim(string(bytesWindowsPath), "\n")

	return
}
