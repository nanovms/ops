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
	env := os.Environ()
	env = append(env, "LD_TRACE_LOADED_OBJECTS=1")
	cmd := exec.Command(bin)
	cmd.Env = env
	out, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(out)
	var deps []string
	cmd.Start()
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		deps = append(deps, strings.TrimSpace(parts[len(parts)-2]))
	}
	defer out.Close()
	cmd.Wait()
	return deps, nil
}
