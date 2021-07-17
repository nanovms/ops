package crossbuild

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// One or more command lines to be executed inside VM.
type vmCommand struct {
	SupressOutput bool
	CommandLines  []string
	Username      string
	Password      string
	Port          int
	OutputBuffer  bytes.Buffer
	ErrorBuffer   bytes.Buffer
}

// CombinedOutput returns combined string of standard output and error.
func (cmd *vmCommand) CombinedOutput() string {
	return cmd.OutputBuffer.String() + cmd.ErrorBuffer.String()
}

// Execute executes command in VM as regular user, or as admin if given asAdmin is true.
func (cmd *vmCommand) Execute() error {
	client, err := newSSHClient(cmd.Port, cmd.Username, cmd.Password)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, cmdLine := range cmd.CommandLines {
		if err := cmd.executeCommand(client, cmdLine); err != nil {
			return err
		}
	}
	return nil
}

// Then adds given command line after last-added one.
func (cmd *vmCommand) Then(command string, args ...interface{}) *vmCommand {
	cmd.CommandLines = append(cmd.CommandLines, formatCommand(command, args...))
	return cmd
}

// AsAdmin modified this command to use admin account.
func (cmd *vmCommand) AsAdmin() *vmCommand {
	cmd.Username = "root"
	cmd.Password = VMRootPassword
	return cmd
}

// Executes a single command.
func (cmd *vmCommand) executeCommand(client *ssh.Client, cmdLine string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = os.Stdin
	if cmd.SupressOutput {
		session.Stdout = &cmd.OutputBuffer
		session.Stderr = &cmd.ErrorBuffer
	} else {
		session.Stdout = io.MultiWriter(os.Stdout, &cmd.OutputBuffer)
		session.Stderr = io.MultiWriter(os.Stderr, &cmd.ErrorBuffer)
	}

	if strings.Contains(cmdLine, "$OPS") {
		cmdLine = strings.ReplaceAll(cmdLine, "$OPS", fmt.Sprintf("/home/%s/.ops/bin/ops", cmd.Username))
	}
	return session.Run(cmdLine)
}

// NewCommand creates new command to be executed using user account.
func (vm *virtualMachine) NewCommand(command string, args ...interface{}) *vmCommand {
	return &vmCommand{
		CommandLines: []string{
			formatCommand(command, args...),
		},
		Username: VMUsername,
		Password: VMUserPassword,
		Port:     vm.ForwardPort,
	}
}

// NewCommandf creates new formatted command to be executed using user account.
func (vm *virtualMachine) NewCommandf(format string, args ...interface{}) *vmCommand {
	return &vmCommand{
		CommandLines: []string{
			fmt.Sprintf(format, args...),
		},
		Username: VMUsername,
		Password: VMUserPassword,
		Port:     vm.ForwardPort,
	}
}

// Formats given command and arguments as a single string.
func formatCommand(command string, args ...interface{}) string {
	line := []interface{}{command}
	line = append(line, args...)
	return fmt.Sprintln(line...)
}
