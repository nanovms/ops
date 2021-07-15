package crossbuild

import (
	"path/filepath"
)

// ExecuteOPS executes ops command with given arguments inside the VM.
func (vm *virtualMachine) ExecuteOPS(args ...interface{}) error {
	opsHomeDirPath := filepath.Join("/home", VMUsername, ".ops")
	executablePath := filepath.Join(opsHomeDirPath, "bin", "ops")
	vmCmd := vm.NewCommand(executablePath, args...)
	return vmCmd.Execute()
}
