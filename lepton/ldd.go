package lepton

import (
	"fmt"
)

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
