package qemu

import (
	"fmt"
	"strconv"
	"strings"
)

// CheckIfVersionSupported checks if current version being used is supported.
func CheckIfVersionSupported() (bool, error) {
	versionWarnMsg := "\nsupported QEMU version is 2.9 or above, please do upgrade.\n"
	version, err := Version()
	if err != nil {
		return false, err
	}
	versionParts := strings.Split(version, ".")

	vMajor, err := strconv.ParseInt(versionParts[0], 10, 32)
	if err != nil {
		return false, err
	}

	if vMajor < 2 {
		fmt.Println(versionWarnMsg)
		return false, err
	}

	if len(versionParts) < 2 {
		fmt.Println(versionWarnMsg)
		return false, err
	}

	vMinor, err := strconv.ParseInt(versionParts[1], 10, 32)
	if err != nil {
		return false, err
	}

	if vMajor == 2 && vMinor < 9 {
		fmt.Println(versionWarnMsg)
		return false, err
	}
	return true, nil
}
