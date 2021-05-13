package lepton

import (
	"bufio"
	"debug/elf"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/constants"
)

// GetElfFileInfo returns an object with elf information of the path program
func GetElfFileInfo(path string) (*elf.File, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	efd, err := elf.NewFile(fd)
	if err != nil {
		return nil, err
	}
	return efd, nil
}

// HasDebuggingSymbols checks whether elf file has debugging symbols
func HasDebuggingSymbols(efd *elf.File) bool {
	for _, phdr := range efd.Sections {
		if strings.Compare(phdr.Name, ".debug_info") == 0 {
			return true
		}
	}

	return false
}

// IsDynamicLinked checks whether elf file was linked dynamically
func IsDynamicLinked(efd *elf.File) bool {
	libs, err := efd.ImportedLibraries()
	if err != nil || len(libs) == 0 {
		return false
	}
	return true
}

// works only on linux, need to
// replace looking up in dynamic section in ELF
func getSharedLibs(targetRoot string, path string) ([]string, error) {
	var notExistLib []string
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Check file is a valid ELF
	isELF, err := isELF(path)
	if err != nil {
		return nil, errors.WrapPrefix(err, path, 0)
	}
	if !isELF {
		log.Fatalf(constants.ErrorColor, "Only ELF binaries are supported. Is thia a Linux binary? run 'file "+path+"' on it\n")
	}

	if _, err := os.Stat(path); err != nil {
		return nil, errors.Wrap(err, 1)
	}
	// LD_TRACE_LOADED_OBJECTS need to fork with out executing it.
	// TODO:move away from LD_TRACE_LOADED_OBJECTS
	dir, _ := os.Getwd()
	var deps []string

	elfFile, err := GetElfFileInfo(path)

	if err == nil && IsDynamicLinked(elfFile) {
		env := os.Environ()
		env = append(env, "LD_TRACE_LOADED_OBJECTS=1")
		cmd := exec.Command(path)
		cmd.Env = env
		out, _ := cmd.StdoutPipe()
		scanner := bufio.NewScanner(out)

		err = cmd.Start()
		if err != nil {
			return deps, err
		}
		// vsdo is user space mapped, there is no backing lib
		vsdo := "linux-vdso.so"
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" || strings.HasPrefix(text, vsdo) {
				continue
			}
			parts := strings.Split(text, " ")
			if strings.HasPrefix(parts[0], vsdo) {
				continue
			}
			libpath, _ := filepath.Abs(parts[len(parts)-2])
			if strings.HasPrefix(libpath, dir) {
				libpath = libpath[len(dir)+1:]
			}
			if strings.Contains(text, "not found") {
				notExistLib = append(notExistLib, text)
			}
			err = errors.New("")
			deps = append(deps, strings.TrimSpace(libpath))
		}

		if len(notExistLib) != 0 {
			errMessage := "Library: "
			for i := range notExistLib {
				errMessage += notExistLib[i]
				if i+1 != len(notExistLib) {
					errMessage += "\n"
				}
			}
			fmt.Println("Ops can't find the following missing libraries:")
			return nil, errors.New(errMessage)
		}

		defer out.Close()
		if err := cmd.Wait(); err != nil {
			return nil, err
		}
	}
	return deps, nil
}

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
