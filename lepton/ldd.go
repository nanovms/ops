package lepton

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// works only on linux, need to
// replace looking up in dynamic section in ELF
func getSharedLibs(path string) ([]string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	var deps []string
	if ok, _ := isDynamicLinked(path); ok {

		env := os.Environ()
		env = append(env, "LD_TRACE_LOADED_OBJECTS=1")
		cmd := exec.Command(path)
		cmd.Env = env
		out, _ := cmd.StdoutPipe()
		scanner := bufio.NewScanner(out)

		cmd.Start()
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
			deps = append(deps, strings.TrimSpace(parts[len(parts)-2]))
		}
		defer out.Close()
		cmd.Wait()
	}
	return deps, nil
}
