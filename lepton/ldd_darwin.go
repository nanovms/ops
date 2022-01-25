package lepton

import (
	"debug/elf"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/constants"
	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/log"
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
	for _, phdr := range efd.Progs {
		if phdr.Type == elf.PT_DYNAMIC {
			return true
		}
	}

	return false
}

func expandVars(origin string, s string) string {
	return strings.Replace(s, "$ORIGIN", origin, -1)
}

// returns the image path and host path
func findLib(targetRoot string, origin string, libDirs []string, path string) (string, string, error) {
	if path[0] == '/' {
		rpath, err := fs.LookupFile(targetRoot, path)
		if err != nil {
			return "", "", err
		}
		return path, rpath, nil
	}

	for _, libDir := range libDirs {
		lib := filepath.Join(expandVars(origin, libDir), path)
		rlib, err := fs.LookupFile(targetRoot, lib)
		if err == nil {
			return lib, rlib, nil
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
	}

	return "", "", os.ErrNotExist
}

func _getSharedLibs(libs map[string]string, targetRoot string, path string) error {
	path, err := fs.LookupFile(targetRoot, path)
	if err != nil {
		return errors.WrapPrefix(err, path, 0)
	}

	fd, err := elf.Open(path)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number") {
			log.Fatalf(constants.ErrorColor, "Only ELF binaries are supported. Is this a Mach-0 (macOS) binary? run 'file "+path+"' on it\n")
		}
		return errors.WrapPrefix(err, path, 0)
	}
	defer fd.Close()

	var libDirs []string

	// 1. LD_LIBRARY_PATH
	var ldLibraryPath []string
	val := os.Getenv("LD_LIBRARY_PATH")
	if len(strings.TrimSpace(val)) > 0 {
		ldLibraryPath = strings.Split(val, ":")
	}

	// 2. DT_RUNPATH
	dtRunpath, err := fd.DynString(elf.DT_RUNPATH)
	if err != nil {
		return err
	}
	if len(dtRunpath) == 0 {
		// DT_RPATH should take precedence over LD_LIBRARY_PATH
		dtRpath, err := fd.DynString(elf.DT_RPATH)
		if err != nil {
			return err
		}
		for _, d := range dtRpath {
			libDirs = append(libDirs, strings.Split(d, ":")...)
		}
		libDirs = append(libDirs, ldLibraryPath...)
	} else {
		libDirs = append(libDirs, ldLibraryPath...)
		for _, d := range dtRunpath {
			libDirs = append(libDirs, strings.Split(d, ":")...)
		}
	}
	libDirs = append(libDirs, "/lib64", "/lib/x86_64-linux-gnu", "/usr/lib", "/usr/lib64", "/usr/lib/x86_64-linux-gnu")

	dtNeeded, err := fd.DynString(elf.DT_NEEDED)
	if err != nil {
		return err
	}

	for _, libpath := range dtNeeded {
		if len(libpath) == 0 {
			continue
		}

		// append library
		libpath, absLibpath, err := findLib(targetRoot, filepath.Dir(path), libDirs, libpath)
		if err != nil {
			return errors.WrapPrefix(err, libpath, 0)
		}
		if _, ok := libs[libpath]; !ok {
			libs[libpath] = absLibpath

			// append library dependencies
			err := _getSharedLibs(libs, targetRoot, absLibpath)
			if err != nil {
				return err
			}
		}
	}

	for _, prog := range fd.Progs {
		if prog.ProgHeader.Type != elf.PT_INTERP {
			continue
		}
		r := prog.Open()
		ipathbuf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		// don't include the terminating NUL in path string
		ipath := string(ipathbuf[:len(ipathbuf)-1])
		ipath, absIpath, err := findLib(targetRoot, filepath.Dir(path), libDirs, ipath)
		if err != nil {
			return errors.WrapPrefix(err, ipath, 0)
		}
		if _, ok := libs[ipath]; !ok {
			libs[ipath] = absIpath
			err := _getSharedLibs(libs, targetRoot, ipath)
			if err != nil {
				return err
			}
		}
		break
	}
	return nil
}

func getSharedLibs(targetRoot string, path string) (map[string]string, error) {
	libs := make(map[string]string)
	err := _getSharedLibs(libs, targetRoot, path)
	if err != nil {
		return nil, err
	}
	return libs, nil
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
