package crossbuild

import (
	"errors"
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
			if dep.Name == "nodejs" || dep.Name == "npm" {
				prepCmd := env.vm.NewCommand(
					"apt-get install curl software-properties-common -y",
				).Then("apt-get update -y").AsAdmin()
				if err := prepCmd.Execute(); err != nil {
					return err
				}

				dep.Name = "nodejs"
			}

			vmCmd := env.vm.NewCommand("apt-get", "install", dep.Name, "-y").AsAdmin()
			if err := vmCmd.Execute(); err != nil {
				return err
			}
			continue
		}

		if dep.Type == "command" {
			vmCmd := env.vm.NewCommandf("cd %s && %s", env.vmDirPath(), dep.Command)
			if dep.AsAdmin {
				vmCmd.AsAdmin()
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

	vmCmd := env.vm.NewCommandf("cd %s && %s", env.vmDirPath(), env.source.Commands.Run)
	return vmCmd.Execute()
}
