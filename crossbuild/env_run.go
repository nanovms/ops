package crossbuild

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nanovms/ops/log"
)

// RunUsingNexe builds given file using nexe and then run the executable.
func (env *Environment) RunUsingNexe(filename string) error {
	if err := env.BuildUsingNexe(filename); err != nil {
		return err
	}

	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGTERM)

	executableName := strings.TrimSuffix(filename, filepath.Ext(filename))
	vmCmd := env.NewCommand(filepath.Join(env.vmSourcePath(), executableName))

	go func() {
		<-interruptSignal
		if err := vmCmd.Terminate(); err != nil {
			log.Warn(err)
		}
	}()

	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return nil
}
