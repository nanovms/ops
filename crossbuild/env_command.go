package crossbuild

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

// EnvironmentCommand is one or more command lines to be executed.
type EnvironmentCommand struct {
	SupressOutput bool
	CommandLines  []string
	Username      string
	Password      string
	Port          int
	OutputBuffer  bytes.Buffer
	ErrorBuffer   bytes.Buffer

	currentSession *ssh.Session
}

// CombinedOutput returns combined string of standard output and error.
func (cmd *EnvironmentCommand) CombinedOutput() string {
	return cmd.OutputBuffer.String() + cmd.ErrorBuffer.String()
}

// Execute executes command in VM as regular user, or as admin if given asAdmin is true.
func (cmd *EnvironmentCommand) Execute() error {
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
func (cmd *EnvironmentCommand) Then(command string, args ...interface{}) *EnvironmentCommand {
	cmd.CommandLines = append(cmd.CommandLines, formatCommand(command, args...))
	return cmd
}

// AsAdmin modified this command to use admin account.
func (cmd *EnvironmentCommand) AsAdmin() *EnvironmentCommand {
	cmd.Username = "root"
	cmd.Password = EnvironmentRootPassword
	return cmd
}

func (cmd *EnvironmentCommand) Terminate() error {
	if cmd.currentSession != nil {
		return cmd.currentSession.Signal(ssh.SIGINT)
	}
	return nil
}

// Executes a single command.
func (cmd *EnvironmentCommand) executeCommand(client *ssh.Client, cmdLine string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd.currentSession = session
	session.Stdin = os.Stdin
	if cmd.SupressOutput {
		session.Stdout = &cmd.OutputBuffer
		session.Stderr = &cmd.ErrorBuffer
	} else {
		session.Stdout = io.MultiWriter(os.Stdout, &cmd.OutputBuffer)
		session.Stderr = io.MultiWriter(os.Stderr, &cmd.ErrorBuffer)
	}
	return session.Run(cmdLine)
}

// Formats given command and arguments as a single string.
func formatCommand(command string, args ...interface{}) string {
	line := []interface{}{command}
	line = append(line, args...)
	return fmt.Sprintln(line...)
}

// NewCommand creates new command to be executed using user account.
func (env *Environment) NewCommand(command string, args ...interface{}) *EnvironmentCommand {
	return &EnvironmentCommand{
		CommandLines: []string{
			formatCommand(command, args...),
		},
		Username: EnvironmentUsername,
		Password: EnvironmentUserPassword,
		Port:     env.Config.Port,
	}
}

// NewCommandf creates new formatted command to be executed using user account.
func (env *Environment) NewCommandf(format string, args ...interface{}) *EnvironmentCommand {
	return env.NewCommand(fmt.Sprintf(format, args...))
}
