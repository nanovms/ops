package lepton

import (
	"debug/elf"
	"errors"
	"github.com/nanovms/ops/types"
)

// GetElfFileInfo returns an object with elf information of the path program
func GetElfFileInfo(path string) (*elf.File, error) {
	return nil, errors.New("un-implemented")
}

// HasDebuggingSymbols checks whether elf file has debugging symbols
func HasDebuggingSymbols(efd *elf.File) bool {
	return false
}

// IsDynamicLinked checks whether elf file was linked dynamically
func IsDynamicLinked(efd *elf.File) bool {
	return false
}

// works only on linux, need to
// replace looking up in dynamic section in ELF
func getSharedLibs(targetRoot string, path string, c *types.Config) (map[string]string, error) {
	return nil, nil
}

// isELF returns true if file is valid ELF
func isELF(path string) (bool, error) {
	return true, errors.New("un-implemented")
}
