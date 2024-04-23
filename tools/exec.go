package tools

import (
	"fmt"
	"os/exec"
	"strings"
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

// ExecCmdList takes a set of strings as a single command and runs it.
func ExecCmdList(str ...string) {
	cmd := exec.Command("bash", "-c", strings.Join(str, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
	}
}
