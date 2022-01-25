package qemu

import (
	"io/ioutil"
	"strings"
)

func hvSupport() (bool, error) {
	const intel string = "EPT"
	const amd string = "POPCNT"

	b, err := ioutil.ReadFile("/var/run/dmesg.boot")
	if err != nil {
		return false, err
	}

	content := string(b)

	if !strings.Contains(content, amd) &&
		!strings.Contains(content, intel) {
		return false, nil
	}

	return true, nil
}
