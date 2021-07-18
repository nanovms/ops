package crossbuild

import (
	"os"
	"path/filepath"
)

// DownloadFile pulls given file from VM's source directory to current directory.
func (env *Environment) DownloadFile(filename string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	destpath := filepath.Join(currentDir, filename)
	srcpath := filepath.Join(env.vmSourcePath(), filename)

	sshClient, err := newSSHClient(env.Config.Port, "root", EnvironmentRootPassword)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	return env.vmDownloadFile(sshClient, srcpath, destpath, 0755)
}

// DownloadExecutableFile pulls given executable file from VM's source directory to bin directory, i.e. $PWD/bin.
func (env *Environment) DownloadExecutableFile(filename string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	binDir := filepath.Join(currentDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	destpath := filepath.Join(binDir, filename)
	srcpath := filepath.Join(env.vmSourcePath(), filename)

	sshClient, err := newSSHClient(env.Config.Port, "root", EnvironmentRootPassword)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	return env.vmDownloadFile(sshClient, srcpath, destpath, 0755)
}
