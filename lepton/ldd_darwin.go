package lepton

import (
	"debug/elf"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
)

func lookupFile(targetRoot string, path string) (string, error) {
	_, err := os.Lstat(path)
	if err == nil || !os.IsNotExist(err) {
		// file exists or other error
		return path, err
	}

	if targetRoot == "" {
		return path, err
	}

	var targetPath string
	currentPath := path
	for {
		targetPath = filepath.Join(targetRoot, currentPath)
		fi, err := os.Lstat(targetPath)
		if err != nil {
			return path, err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			break
		}

		currentPath, err = os.Readlink(targetPath)
		if err != nil {
			return path, err
		}
		if currentPath[0] != '/' {
			// relative symlinks are ok
			break
		}
	}

	// fmt.Printf("%s -> %s\n", path, targetPath)
	return targetPath, nil
}

func findLib(targetRoot string, libDirs []string, path string) (string, error) {
	if path[0] == '/' {
		if _, err := lookupFile(targetRoot, path); err != nil {
			return "", err
		}
		return path, nil
	}

	for _, libDir := range libDirs {
		lib := filepath.Join(libDir, path)
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
		return nil, errors.WrapPrefix(err, path, 0)
	}
	defer fd.Close()

	dt_runpath, err := fd.DynString(elf.DT_RUNPATH)
	if err != nil {
		return nil, err
	}
	if len(dt_runpath) == 0 {
		if dt_runpath, err = fd.DynString(elf.DT_RPATH); err != nil {
			return nil, err
		}
	}
	dt_runpath = append(dt_runpath, "/lib64", "/lib/x86_64-linux-gnu", "/usr/lib", "/usr/lib64", "/usr/lib/x86_64-linux-gnu")

	val := os.Getenv("LD_LIBRARY_PATH")
	if len(strings.TrimSpace(val)) > 0 {
		dt_runpath = append(dt_runpath, val)
	}

	dt_needed, err := fd.DynString(elf.DT_NEEDED)
	if err != nil {
		return nil, err
	}

	var libs []string
	for _, libpath := range dt_needed {
		if len(libpath) == 0 {
			continue
		}

		// append library
		absLibpath, err := findLib(targetRoot, dt_runpath, libpath)
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
	for key, _ := range keys {
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
