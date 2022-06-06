//go:build darwin
// +build darwin

package qemu

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// I'm not sure this is even necessary anymore.
// From what I've read you basically need a mac from 2010 to have this
// return false.
func hvSupport() (bool, error) {

	cmd := exec.Command("/usr/sbin/sysctl", "kern.hv_support")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return false, err
	}

	if strings.Contains(out.String(), "1") {
		return true, nil
	}

	return false, err
}
