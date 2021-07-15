package crossbuild

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Represents a command to be executed inside VM.
type virtualMachineCommand struct {
	SupressOutput bool

	command      string
	arguments    []interface{}
	port         int
	outputBuffer bytes.Buffer
	errorBuffer  bytes.Buffer
}

// CombinedOutput returns combined string of standard output and error.
func (cmd *virtualMachineCommand) CombinedOutput() string {
	return cmd.outputBuffer.String() + cmd.errorBuffer.String()
}

// Execute executes command in VM as regular user.
func (cmd *virtualMachineCommand) Execute() error {
	return cmd.executeCommand(VMUsername, VMUserPassword)
}

// ExecuteAsSuperUser executes command in VM as super user.
func (cmd *virtualMachineCommand) ExecuteAsSuperUser() error {
	return cmd.executeCommand("root", VMRootPassword)
}

// Executes command inside the VM given the SSH client.
// TODO: how to handle app with main loop ? catch ctr-C and send sigint to process ?
func (cmd *virtualMachineCommand) executeCommand(username, password string) error {
	client, err := newSSHClient(cmd.port, username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	commandLine := cmd.command
	for _, arg := range cmd.arguments {
		commandLine += fmt.Sprintf(" %v", arg)
	}

	if strings.Contains(commandLine, "$OPS") {
		commandLine = strings.ReplaceAll(commandLine, "$OPS", fmt.Sprintf("/home/%s/.ops/bin/ops", VMUsername))
	}

	session.Stdin = os.Stdin
	if cmd.SupressOutput {
		session.Stdout = &cmd.outputBuffer
		session.Stderr = &cmd.errorBuffer
	} else {
		session.Stdout = io.MultiWriter(os.Stdout, &cmd.outputBuffer)
		session.Stderr = io.MultiWriter(os.Stderr, &cmd.errorBuffer)
	}

	if err = session.Run(commandLine); err != nil {
		return err
	}
	return nil
}

// NewCommand creates new command.
func (vm *virtualMachine) NewCommand(command string, args ...interface{}) *virtualMachineCommand {
	return &virtualMachineCommand{
		command:   command,
		arguments: args,
		port:      vm.ForwardPort,
	}
}

// Creates new ssh client connected to given port using given credentials.
func newSSHClient(port int, username, password string) (*ssh.Client, error) {
	return ssh.Dial("tcp", fmt.Sprintf("localhost:%v", port), &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}
