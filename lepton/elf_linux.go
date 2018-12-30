package lepton

import (
	"debug/elf"
	"os"
)

func isDynamicLinked(path string) (bool, error) {
	fd, err := os.Open(path)
	if err != nil {
		return false, err
	}
	efd, err := elf.NewFile(fd)
	if err != nil {
		return false, err
	}
	for _, phdr := range efd.Progs {
		if phdr.Type == elf.PT_DYNAMIC {
			return true, nil
		}
	}

	return false, nil
}
