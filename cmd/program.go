package cmd

import (
	"os"
	"path"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

func checkProgramExists(program string) {
	_, err := os.Stat(path.Join(api.GetOpsHome(), program))
	_, err1 := os.Stat(program)

	if os.IsNotExist(err) && os.IsNotExist(err1) {
		log.Fatalf("error: %v: %v\n", program, err)
	}
}
