package crossbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// OPSImageFileExtension is extension name of unikernel image.
	OPSImageFileExtension = ".img"
)

var (
	// OPSExecutablePath is path to OPS executable.
	OPSExecutablePath = filepath.Join("/home", EnvironmentUsername, ".ops", "bin", "ops")

	// OPSImageDirPath is path to OPS image directory.
	OPSImageDirPath = filepath.Join("/home", EnvironmentUsername, ".ops", "images")
)

// NewOpsCommand creates new command that executes ops executable.
func (env *Environment) NewOpsCommand(args ...interface{}) *EnvironmentCommand {
	return env.NewCommandf("cd %s && %s %s", env.vmSourcePath(), OPSExecutablePath, fmt.Sprintln(args...))
}

// OpsBuildImage builds unikernel image using OPS.
func (env *Environment) OpsBuildImage(executablePath string) error {
	nameParts := strings.Split(filepath.Base(executablePath), ".")
	executableName := nameParts[0]
	vmCmd := env.NewOpsCommand("build", executablePath, "--imagename", executableName)
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	return env.OpsDownloadImage(executableName)
}

// OpsDownloadImage pulls unikernel image given its name.
func (env *Environment) OpsDownloadImage(name string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	imageName := name + OPSImageFileExtension
	destpath := filepath.Join(currentDir, "bin", imageName)
	srcpath := filepath.Join(OPSImageDirPath, imageName)

	sshClient, err := newSSHClient(env.Config.Port, "root", EnvironmentRootPassword)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	return env.vmDownloadFile(sshClient, srcpath, destpath, 0755)
}
