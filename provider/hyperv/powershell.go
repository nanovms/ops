//go:build hyperv || !onlyprovider

package hyperv

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nanovms/ops/wsl"
)

const (
	powerShellFalse = "False"
	powerShellTrue  = "True"
)

// PowerShellCmd executes powershell commands
type PowerShellCmd struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Run executes powershell command
func (ps *PowerShellCmd) Run(fileContents string, params ...string) error {
	_, err := ps.Output(fileContents, params...)
	return err
}

// Output runs the PowerShell command and returns its standard output.
func (ps *PowerShellCmd) Output(fileContents string, params ...string) (string, error) {
	path, err := ps.getPowerShellPath()
	if err != nil {
		return "", err
	}

	filename, err := saveScript(fileContents)
	if err != nil {
		return "", err
	}

	debug := os.Getenv("PACKER_POWERSHELL_DEBUG") != ""
	verbose := debug || os.Getenv("PACKER_POWERSHELL_VERBOSE") != ""

	if !debug {
		defer os.Remove(filename)
	}

	args := createArgs(filename, params...)

	if verbose {
		log.Printf("Run: %s %s", path, args)
	}

	var stdout, stderr bytes.Buffer
	command := exec.Command(path, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()

	if ps.Stdout != nil {
		stdout.WriteTo(ps.Stdout)
	}

	if ps.Stderr != nil {
		stderr.WriteTo(ps.Stderr)
	}

	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("PowerShell error: %s", stderrString)
	}

	if len(stderrString) > 0 {
		err = fmt.Errorf("PowerShell error: %s", stderrString)
	}

	stdoutString := strings.TrimSpace(stdout.String())

	if verbose && stdoutString != "" {
		log.Printf("stdout: %s", stdoutString)
	}

	// only write the stderr string if verbose because
	// the error string will already be in the err return value.
	if verbose && stderrString != "" {
		log.Printf("stderr: %s", stderrString)
	}

	return stdoutString, err
}

// IsPowershellAvailable checks whether powershell executable exists in os
func IsPowershellAvailable() (bool, string, error) {
	path, err := exec.LookPath("powershell")
	if err == nil {
		return true, path, err
	}

	path, err = exec.LookPath("powershell.exe")
	if err == nil {
		return true, path, err
	}

	return false, "", err
}

func (ps *PowerShellCmd) getPowerShellPath() (string, error) {
	powershellAvailable, path, err := IsPowershellAvailable()

	if !powershellAvailable {
		log.Fatalf("Cannot find PowerShell in the path")
		return "", err
	}

	return path, nil
}

func saveScript(fileContents string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "ps")
	if err != nil {
		return "", err
	}

	_, err = file.Write([]byte(fileContents))
	if err != nil {
		return "", err
	}

	err = file.Close()
	if err != nil {
		return "", err
	}

	newFilename := file.Name() + ".ps1"
	err = os.Rename(file.Name(), newFilename)
	if err != nil {
		return "", err
	}

	scriptWindowsPath, err := wsl.ConvertPathFromWSLtoWindows(newFilename)
	if err != nil {
		return "", err
	}

	return scriptWindowsPath, nil
}

func createArgs(filename string, params ...string) []string {
	args := make([]string, len(params)+5)
	args[0] = "-ExecutionPolicy"
	args[1] = "Bypass"

	args[2] = "-NoProfile"

	args[3] = "-File"
	args[4] = filename

	for key, value := range params {
		args[key+5] = value
	}

	return args
}

// getHostAvailableMemory returns the host available memory
func getHostAvailableMemory() float64 {

	var script = "(Get-WmiObject Win32_OperatingSystem).FreePhysicalMemory / 1024"

	var ps PowerShellCmd
	output, _ := ps.Output(script)

	freeMB, _ := strconv.ParseFloat(output, 64)

	return freeMB
}

func getHostName(ip string) (string, error) {

	var script = `
param([string]$ip)
try {
  $HostName = [System.Net.Dns]::GetHostEntry($ip).HostName
  if ($HostName -ne $null) {
    $HostName = $HostName.Split('.')[0]
  }
  $HostName
} catch { }
`

	//
	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, ip)
	if err != nil {
		return "", err
	}

	return cmdOut, nil
}

func isCurrentUserAnAdministrator() (bool, error) {
	var script = `
$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
$principal = new-object System.Security.Principal.WindowsPrincipal($identity)
$administratorRole = [System.Security.Principal.WindowsBuiltInRole]::Administrator
return $principal.IsInRole($administratorRole)
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script)
	if err != nil {
		return false, err
	}

	res := strings.TrimSpace(cmdOut)
	return res == powerShellTrue, nil
}

func moduleExists(moduleName string) (bool, error) {

	var script = `
param([string]$moduleName)
(Get-Module -Name $moduleName) -ne $null
`
	var ps PowerShellCmd
	cmdOut, err := ps.Output(script)
	if err != nil {
		return false, err
	}

	res := strings.TrimSpace(string(cmdOut))

	if res == powerShellFalse {
		err := fmt.Errorf("powerShell %s module is not loaded. Make sure %s feature is on", moduleName, moduleName)
		return false, err
	}

	return true, nil
}
