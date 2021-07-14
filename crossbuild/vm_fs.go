package crossbuild

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
)

// MkdirAll executes 'mkdir -p' command in VM.
func (vm *virtualMachine) MkdirAll(dirpath string) error {
	vmCmd := vm.NewCommand("mkdir", "-p", dirpath)
	return vmCmd.Execute()
}

// RemoveDir removed directory recursively given its path.
func (vm *virtualMachine) RemoveDir(dirpath string) error {
	vmCmd := vm.NewCommand("rm", "-rf", dirpath)
	return vmCmd.Execute()
}

// CopyFile copies file from srcpath in local filesystem to destpath in VM.
func (vm *virtualMachine) CopyFile(client *ssh.Client, srcpath, destpath string, permissions fs.FileMode) error {
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
