package lepton

import (
	"debug/elf"
	"errors"
)

// IsDynamicLinked stub
func IsDynamicLinked(efd *elf.File) bool {
	return false
}

// HasDebuggingSymbols cstub
func HasDebuggingSymbols(efd *elf.File) bool {
	return false
}

// GetElfFileInfo stub
func GetElfFileInfo(path string) (*elf.File, error) {
	return nil, errors.New("unsupported")
}

// stub
func getSharedLibs(targetRoot string, path string) ([]string, error) {
	var deps []string
	return deps, nil
}
