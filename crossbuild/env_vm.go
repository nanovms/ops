package crossbuild

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
)

// Executes 'mkdir -p' in VM.
func (env *Environment) vmMkdirAll(dirpath string) error {
	vmCmd := env.NewCommand("mkdir", "-p", dirpath)
	return vmCmd.Execute()
}

// Recursively removes given directory in VM.
func (env *Environment) vmRemoveDir(dirpath string) error {
	vmCmd := env.NewCommand("rm", "-rf", dirpath)
	return vmCmd.Execute()
}

// Copies file from srcpath in local filesystem to destpath in VM.
func (env *Environment) vmCopyFile(client *ssh.Client, srcpath, destpath string, permissions fs.FileMode) error {
	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return err
	}
	defer scpClient.Close()

	file, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer file.Close()

	perm := fmt.Sprintf("%04o", uint32(permissions))
	if err := scpClient.CopyFile(file, destpath, perm); err != nil {
		return err
	}
	return nil
}

// Downloads file from srcpath in VM to local filesystem.
func (env *Environment) vmDownloadFile(client *ssh.Client, srcpath, destpath string, permissions fs.FileMode) error {
	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return err
	}
	defer scpClient.Close()

	file, err := os.OpenFile(destpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, permissions)
	if err != nil {
		return err
	}
	defer file.Close()

	return scpClient.CopyFromRemote(file, srcpath)
}

// Returns path to environment directory inside VM.
func (env *Environment) vmSourcePath() string {
	return filepath.Join("/home", EnvironmentUsername, "sources", env.WorkingDirectory)
}
