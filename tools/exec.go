package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

func ExecCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	output = string(out)
	return
}

func ExecCmdList(str ...string) {
	cmd := exec.Command("/bin/bash", "-c", strings.Join(str, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(out))
	}
}
