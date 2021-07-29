package crossbuild

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// EnvironmentCommand is one or more command lines to be executed.
type EnvironmentCommand struct {
	SuppressOutput bool
	CommandLines   []string
	Username       string
	Password       string
	Port           int
	OutputBuffer   bytes.Buffer
	ErrorBuffer    bytes.Buffer
	currentSession *ssh.Session
}

// StdOutput returns the string written to the standard output stream by a command.
func (cmd *EnvironmentCommand) StdOutput() string {
	return cmd.OutputBuffer.String()
}

// StdError returns the string written to the standard error stream by a command.
func (cmd *EnvironmentCommand) StdError() string {
	return cmd.ErrorBuffer.String()
}

// Execute executes command in VM as regular user, or as admin if asAdmin is true.
func (cmd *EnvironmentCommand) Execute() error {
	client, err := newSSHClient(cmd.Port, cmd.Username, cmd.Password)
	if err != nil {
		return err
	}
	defer client.Close()
	for _, cmdLine := range cmd.CommandLines {
		if err := cmd.executeCommand(client, cmdLine); err != nil {
			if cmd.SuppressOutput {
				return errors.New(strings.TrimSpace(cmd.StdError()))
			}
			return err
		}
	}
	return nil
}

// Then appends command line to the set of commands.
func (cmd *EnvironmentCommand) Then(command string, args ...interface{}) *EnvironmentCommand {
	cmd.CommandLines = append(cmd.CommandLines, formatCommand(command, args...))
	return cmd
}

// AsAdmin modifies the command to use admin account.
func (cmd *EnvironmentCommand) AsAdmin() *EnvironmentCommand {
	cmd.Username = "root"
	cmd.Password = EnvironmentRootPassword
	return cmd
}

// Executes a single command line.
func (cmd *EnvironmentCommand) executeCommand(client *ssh.Client, cmdLine string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd.currentSession = session
	session.Stdin = os.Stdin
	if cmd.SuppressOutput {
		session.Stdout = &cmd.OutputBuffer
		session.Stderr = &cmd.ErrorBuffer
	} else {
		session.Stdout = io.MultiWriter(os.Stdout, &cmd.OutputBuffer)
		session.Stderr = io.MultiWriter(os.Stderr, &cmd.ErrorBuffer)
	}
	return session.Run(cmdLine)
}

// Formats command and arguments as a single string.
func formatCommand(command string, args ...interface{}) string {
	line := []interface{}{command}
	line = append(line, args...)
	return fmt.Sprintln(line...)
}
