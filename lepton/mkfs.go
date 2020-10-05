package lepton

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	errMKFSSetupCommandRequired = fmt.Errorf("SetupCommand must run before")
)

// MkfsCommand wraps mkfs calls
type MkfsCommand struct {
	binaryPath string
	args       []string
	stdin      *os.File
	output     []byte
	command    *exec.Cmd
}

// NewMkfsCommand returns an instance of MkfsCommand
func NewMkfsCommand(binaryPath string) *MkfsCommand {
	args := []string{}

	return &MkfsCommand{
		binaryPath: binaryPath,
		args:       args,
		stdin:      nil,
		command:    nil,
	}
}

// SetupCommand instantiates a command with the args assigned
func (m *MkfsCommand) SetupCommand() {
	m.command = exec.Command(m.binaryPath, m.args...)
	if m.stdin != nil {
		m.command.Stdin = m.stdin
	}
}

// SetEmptyFileSystem add argument that sets file system as empty
func (m *MkfsCommand) SetEmptyFileSystem() {
	m.args = append(m.args, "-e")
}

// SetFileSystemSize add argument that sets file system size
func (m *MkfsCommand) SetFileSystemSize(size string) {
	m.args = append(m.args, "-s", size)
}

// SetTargetRoot add argument that sets
func (m *MkfsCommand) SetTargetRoot(targetRoot string) {
	m.args = append(m.args, "-r", targetRoot)
}

// SetBoot add argument that sets
func (m *MkfsCommand) SetBoot(boot string) {
	m.args = append(m.args, "-b", boot)
}

// SetFileSystemPath add argument that sets file system path
func (m *MkfsCommand) SetFileSystemPath(fsPath string) {
	m.args = append(m.args, fsPath)
}

// SetStdin sets process's standard input
func (m *MkfsCommand) SetStdin(file *os.File) {
	m.stdin = file
}

// Execute runs mkfs command
func (m *MkfsCommand) Execute() error {
	if m.command == nil {
		return errMKFSSetupCommandRequired
	}

	out, err := m.command.CombinedOutput()

	m.output = out

	return err
}

// GetStdinPipe returns a pipe that will be connected to the command's standard input when the command starts.
func (m *MkfsCommand) GetStdinPipe() (io.WriteCloser, error) {
	if m.command == nil {
		return nil, errMKFSSetupCommandRequired
	}

	return m.command.StdinPipe()
}

// GetOutput returns command execution output
func (m *MkfsCommand) GetOutput() []byte {
	return m.output
}

// GetArgs returns the arguments assigned to command
func (m *MkfsCommand) GetArgs() []string {
	return m.args
}

// GetUUID returns the uuid of file system built
func (m *MkfsCommand) GetUUID() string {
	return uuidFromMKFS(m.output)
}

// uuidFromMKFS reads the resulting UUID from nanos mkfs tool
func uuidFromMKFS(b []byte) string {
	var uuid string
	in := string(b)
	fields := strings.Fields(in)
	for i, f := range fields {
		if strings.Contains(f, "UUID") {
			uuid = fields[i+1]
		}
	}
	return strings.TrimSpace(uuid)
}
