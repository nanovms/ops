package lepton

import (
	"debug/elf"
	"errors"
	"github.com/nanovms/ops/types"
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
func getSharedLibs(targetRoot string, path string, c *types.Config) (map[string]string, error) {
	var deps = make(map[string]string)
	return deps, nil
}
