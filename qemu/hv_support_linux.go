package qemu

import (
	"os"
	"strings"
)

func hvSupport() (bool, error) {
	if isInception() {
		return true, nil
	}

	const intel string = "vmx"
	const amd string = "svm"

	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false, err
	}

	content := string(b)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		kvp := strings.Split(line, ":")
		if len(kvp) == 0 {
			continue
		}
		if strings.TrimSpace(kvp[0]) != "flags" {
			continue
		}
		for _, flag := range strings.Split(kvp[1], " ") {
			if flag == intel || flag == amd {
				return true, nil
			}
			continue
		}
	}

	return false, nil
}
