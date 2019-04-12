package lepton

import (
	"bufio"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// "github.com/go-errors/errors"
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

// works only on linux, need to
// replace looking up in dynamic section in ELF
func getSharedLibs(targetRoot string, path string) ([]string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	// LD_TRACE_LOADED_OBJECTS need to fork with out executing it.
	// TODO:move away from LD_TRACE_LOADED_OBJECTS

	// Why this chmod exists? It's failing when program is from something like /bin folder
	// err = os.Chmod(path, 0775)
	// if err != nil {
	// 	return nil, errors.Wrap(err, 1)
	// }

	dir, _ := os.Getwd()
	var deps []string
	if ok, _ := isDynamicLinked(path); ok {

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
			deps = append(deps, strings.TrimSpace(libpath))
		}
		defer out.Close()
		cmd.Wait()
	}
	return deps, nil
}
