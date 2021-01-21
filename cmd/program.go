package cmd

import (
	"fmt"
	"os"
	"path"

	api "github.com/nanovms/ops/lepton"
)

func checkProgramExists(program string) {
	_, err := os.Stat(path.Join(api.GetOpsHome(), program))
	_, err1 := os.Stat(program)

	if os.IsNotExist(err) && os.IsNotExist(err1) {
		fmt.Fprintf(os.Stderr, "error: %v: %v\n", program, err)
		os.Exit(1)
	}
}
