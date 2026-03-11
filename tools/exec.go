package tools

import (
	"os/exec"
)

// ExecCmd takes a command string and runs it.
func ExecCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	output = string(out)
	return
}
