package crossbuild

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// Uses ps command to check if process with given pid running.
func isProcessAlive(pid int) (bool, error) {
	cmd := exec.Command("ps", "ax")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	pidStr := strconv.Itoa(pid)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ")
		if strings.HasPrefix(line, pidStr) {
			return true, nil
		}
	}
	return false, nil
}
