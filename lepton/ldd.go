package lepton

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// works only on linux, need to
// replace looking up in dynamic section in ELF
func getSharedLibs(path string) ([]string, error) {
	bin, err := exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	var deps []string
	if ok, _ := isDynamicLinked(path); ok {

		env := os.Environ()
		env = append(env, "LD_TRACE_LOADED_OBJECTS=1")
		cmd := exec.Command(bin)
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
