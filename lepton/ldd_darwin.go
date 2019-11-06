package lepton

import (
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
)

func expandVars(origin string, s string) string {
	return strings.Replace(s, "$ORIGIN", origin, -1)
}

func findLib(targetRoot string, origin string, libDirs []string, path string) (string, error) {
	if path[0] == '/' {
		if _, err := lookupFile(targetRoot, path); err != nil {
			return "", err
		}
		return path, nil
	}

	for _, libDir := range libDirs {
		lib := filepath.Join(expandVars(origin, libDir), path)
		_, err := lookupFile(targetRoot, lib)
		if err == nil {
			return lib, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", os.ErrNotExist
}

func _getSharedLibs(targetRoot string, path string) ([]string, error) {
	path, err := lookupFile(targetRoot, path)
	if err != nil {
		return nil, errors.WrapPrefix(err, path, 0)
	}

	fd, err := elf.Open(path)
	if err != nil {
		if strings.Contains(err.Error(), "bad magic number") {
			fmt.Printf(ErrorColor, "Only ELF binaries are supported. Is thia a Mach-0 (osx) binary? run 'file "+path+"' on it\n")
			os.Exit(1)
		}
		return nil, errors.WrapPrefix(err, path, 0)
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
		return nil, err
	}
	if len(dtRunpath) == 0 {
		// DT_RPATH should take precedence over LD_LIBRARY_PATH
		dtRpath, err := fd.DynString(elf.DT_RPATH)
		if err != nil {
			return nil, err
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
		return nil, err
	}

	var libs []string
	for _, libpath := range dtNeeded {
		if len(libpath) == 0 {
			continue
		}

		// append library
		absLibpath, err := findLib(targetRoot, filepath.Dir(path), libDirs, libpath)
		if err != nil {
			return nil, errors.WrapPrefix(err, libpath, 0)
		}
		libs = append(libs, absLibpath)

		// append library dependencies
		deplibs, err := _getSharedLibs(targetRoot, absLibpath)
		if err != nil {
			return nil, err
		}
		libs = append(libs, deplibs...)
	}

	return libs, nil
}

func unique(a []string) []string {
	keys := map[string]bool{}
	for v := range a {
		keys[a[v]] = true
	}

	result := []string{}
	for key := range keys {
		result = append(result, key)
	}
	return result
}

func getSharedLibs(targetRoot string, path string) ([]string, error) {
	libs, err := _getSharedLibs(targetRoot, path)
	if err != nil {
		return nil, err
	}
	return unique(libs), nil
}
