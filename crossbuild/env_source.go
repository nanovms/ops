package crossbuild

import (
	"errors"
	"fmt"
)

var (
	ErrNoDependenciesConfigured = errors.New("there are no dependencies configured")
	ErrRunCommandNotConfigured  = errors.New("run command is not configured")
)

// InstallDependencies installs configured source dependencies in VM.
func (env *Environment) InstallDependencies() error {
	if len(env.source.Dependencies) == 0 {
		return ErrNoDependenciesConfigured
	}

	for _, dep := range env.source.Dependencies {
		if dep.Type == "package" {
			vmCmd := env.vm.NewCommand("apt-get", "install", dep.Name, "-y")
			if err := vmCmd.ExecuteAsSuperUser(); err != nil {
				return err
			}
			continue
		}

		if dep.Type == "command" {
			vmCmd := env.vm.NewCommand(dep.Command)
			if dep.AsAdmin {
				if err := vmCmd.ExecuteAsSuperUser(); err != nil {
					return err
				}
				continue
			}

			if err := vmCmd.Execute(); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

// Run executes run command inside VM.
func (env *Environment) Run() error {
	if env.source.Commands.Run == "" {
		return ErrRunCommandNotConfigured
	}

	if err := env.Sync(); err != nil {
		return err
	}

	vmCmd := env.vm.NewCommand(fmt.Sprintf("cd %s && %s", env.vmDirPath(), env.source.Commands.Run))
	return vmCmd.Execute()
}
