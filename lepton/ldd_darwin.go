package lepton

import (
	"debug/elf"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
)

func lookupFile(targetRoot string, path string) (string, error) {
	if targetRoot != "" {
		var targetPath string
		currentPath := path
		for {
			targetPath = filepath.Join(targetRoot, currentPath)
			fi, err := os.Lstat(targetPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return path, err
				}
				// lookup on host
				break
			}
			if fi.Mode()&os.ModeSymlink == 0 {
				// not a symlink found in target root
				return targetPath, nil
			}

			currentPath, err = os.Readlink(targetPath)
			if err != nil {
				return path, err
			}
			if currentPath[0] != '/' {
				// relative symlinks are ok
				path = targetPath
				break
			}

			// absolute symlinks need to be resolved again
		}
	}

	_, err := os.Stat(path)
	return path, err
}

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
		return nil, errors.WrapPrefix(err, path, 0)
	}
	defer fd.Close()

	var libDirs []string

	// 1. LD_LIBRARY_PATH
	var ld_library_path []string
	val := os.Getenv("LD_LIBRARY_PATH")
	if len(strings.TrimSpace(val)) > 0 {
		ld_library_path = strings.Split(val, ":")
	}

	// 2. DT_RUNPATH
	dt_runpath, err := fd.DynString(elf.DT_RUNPATH)
	if err != nil {
		return nil, err
	}
	if len(dt_runpath) == 0 {
		// DT_RPATH should take precedence over LD_LIBRARY_PATH
		dt_rpath, err := fd.DynString(elf.DT_RPATH)
		if err != nil {
			return nil, err
		}
		for _, d := range dt_rpath {
			libDirs = append(libDirs, strings.Split(d, ":")...)
		}
		libDirs = append(libDirs, ld_library_path...)
	} else {
		libDirs = append(libDirs, ld_library_path...)
		for _, d := range dt_runpath {
			libDirs = append(libDirs, strings.Split(d, ":")...)
		}
	}
	libDirs = append(libDirs, "/lib64", "/lib/x86_64-linux-gnu", "/usr/lib", "/usr/lib64", "/usr/lib/x86_64-linux-gnu")

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
