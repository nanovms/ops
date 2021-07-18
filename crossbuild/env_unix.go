package crossbuild

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nanovms/ops/qemu"
)

// Checks environment's process ID, if running.
func environmentProcessID(env *Environment) (int, error) {
	cmdPS := exec.Command("ps", "ax")
	cmdGrep := exec.Command("grep", "qemu")

	reader, writer := io.Pipe()
	cmdPS.Stdout = writer
	cmdGrep.Stdin = reader

	defer reader.Close()

	var buff bytes.Buffer
	cmdGrep.Stdout = &buff

	if err := cmdPS.Start(); err != nil {
		return 0, err
	}

	if err := cmdGrep.Start(); err != nil {
		return 0, err
	}

	if err := cmdPS.Wait(); err != nil {
		return 0, err
	}

	if err := writer.Close(); err != nil {
		return 0, err
	}

	if err := cmdGrep.Wait(); err != nil {
		return 0, err
	}

	output := buff.String()
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ")
		if strings.Contains(line, qemu.BaseCommand) &&
			strings.Contains(line, fmt.Sprintf("-drive file=%s", filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension))) {

			parts := strings.Split(line, " ")
			pid, err := strconv.ParseInt(parts[0], 10, 32)
			if err != nil {
				return 0, err
			}
			return int(pid), nil
		}
	}
	return 0, nil
}
