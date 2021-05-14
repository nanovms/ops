package lepton

import (
	"debug/elf"
	"fmt"
	"strings"
)

// isELF returns true if file is valid ELF
func isELF(path string) (bool, error) {
	fd, err := elf.Open(path)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number") {
			return false, nil
		}
		return false, err
	}
	fd.Close()
	return true, nil
}

// ValidateELF validates ELF executable format given the file path
func ValidateELF(executablePath string) {
	valid, err := isELF(executablePath)
	if err != nil {
		exitWithError(err.Error())
	}
	if !valid {
		exitWithError(fmt.Sprintf(`only ELF binaries are supported. Is thia a Linux binary? run "file %s" on it`, executablePath))
	}
}
