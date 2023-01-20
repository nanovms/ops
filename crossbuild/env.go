package crossbuild

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"
	"github.com/tj/go-spin"
	"golang.org/x/crypto/ssh"
)

const (
	// EnvironmentDownloadBaseURL is the base url to download environment image.
	EnvironmentDownloadBaseURL = "storage.googleapis.com/nanos-build-envs"

	// EnvironmentImageFileExtension is extension for environment image file.
	EnvironmentImageFileExtension = ".qcow2"

	// EnvironmentUsername is username for regular account.
	EnvironmentUsername = "user"

	// EnvironmentUserPassword is password for regular account.
	EnvironmentUserPassword = "user"

	// EnvironmentRootPassword is password for admin account.
	EnvironmentRootPassword = "root"
)

// Environment is a virtualized crossbuild operating system.
type Environment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	IsDefault bool   `json:"is_default"`
	SSHPort   int    `json:"-"`
	PID       int    `json:"-"`
}

// IsInstalled returns whether the environment is installed.
func (env *Environment) IsInstalled() bool {
	imageFilePath := filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension)
	info, err := os.Stat(imageFilePath)
	if err != nil {
		return false
	}
	if info.Mode().IsRegular() {
		return true
	}
	return false
}

// Install pulls environment image from remote storage.
func (env *Environment) Install() error {
	compressedImgFileName := env.ID + ".zip"
	compressedImgFilePath := filepath.Join(EnvironmentImageDirPath, compressedImgFileName)
	downloadURL := "https://" + path.Join(EnvironmentDownloadBaseURL, compressedImgFileName)
	if err := lepton.DownloadFileWithProgress(compressedImgFilePath, downloadURL, 1800); err != nil {
		return err
	}
	defer os.Remove(compressedImgFilePath)
	zipReader, err := zip.OpenReader(compressedImgFilePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()
	for _, file := range zipReader.File {
		if !file.FileInfo().Mode().IsRegular() {
			continue
		}
		targetPath := filepath.Join(EnvironmentImageDirPath, file.Name)
		targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer targetFile.Close()
		zFile, err := file.Open()
		if err != nil {
			return err
		}
		defer zFile.Close()
		log.Infof("Extracting file %s", file.Name)
		if _, err := io.Copy(targetFile, zFile); err != nil {
			return err
		}
	}
	return nil
}

// Boot starts environment VM.
func (env *Environment) Boot() error {
	if env.PID == 0 {
		hypervisor := qemu.HypervisorInstance()
		if hypervisor == nil {
			return errors.New("no hypervisor found in PATH")
		}
		env.SSHPort = 1024
		for env.SSHPort < 0xFFFF { // Look for a free TCP port
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%v", env.SSHPort))
			if err != nil {
				break
			} else {
				conn.Close()
				env.SSHPort++
			}
		}
		cmd := hypervisor.Command(&types.RunConfig{
			Accel:     true,
			Imagename: filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension),
			Vga:       true,
			Memory:    "2G",
			Ports: []string{
				fmt.Sprintf("%d-%d", env.SSHPort, 22),
			},
		})
		if err := cmd.Start(); err != nil {
			return err
		}
		if cmd.Process == nil {
			return errors.New("boot failure")
		}
		env.PID = cmd.Process.Pid
	}
	var (
		wg              = &sync.WaitGroup{}
		timedOut        = false
		checkCount      = 0
		keepChecking    = true
		interruptSignal = make(chan os.Signal, 1)
		interrupted     = false
	)
	wg.Add(1)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interruptSignal
		interrupted = true
		keepChecking = false
	}()
	spinner := spin.New()
	go func() {
		for {
			fmt.Printf("\r%s Starting environment", spinner.Next())
			time.Sleep(time.Second)
			if !keepChecking {
				break
			}
			checkCount++
			if checkCount >= 10 {
				if env.IsReady() {
					fmt.Print("   OK")
					break
				}
				if checkCount >= 60 {
					timedOut = true
					break
				}
			}
		}
		fmt.Println("")
		wg.Done()
	}()
	wg.Wait()
	if interrupted {
		return errors.New("interrupted by user")
	}
	if timedOut {
		return errors.New("not responding")
	}
	return nil
}

// IsReady returns true if communication with environment VM can be established.
func (env *Environment) IsReady() bool {
	vmCmd := env.NewCommand("ls /")
	vmCmd.SuppressOutput = true
	if err := vmCmd.Execute(); err != nil {
		return false
	}
	return vmCmd.StdOutput() != ""
}

// RunCommands runs a sequence of commands in the VM.
func (env *Environment) RunCommands(cmdFile string) error {
	contents, err := os.ReadFile(cmdFile)
	if err != nil {
		return fmt.Errorf("cannot read command file: %v", err)
	}
	vmCmd := env.NewEmptyCommand()
	for _, cmd := range strings.Split(string(contents), "\n") {
		vmCmd = vmCmd.Then(cmd)
	}
	if err := vmCmd.AsAdmin().Execute(); err != nil {
		return errors.New("cannot execute build commands")
	}
	return nil
}

// GetFiles copies image files from VM to local directory.
func (env *Environment) GetFiles(c *types.Config, hostPath string) error {
	sshClient, err := newSSHClient(env.SSHPort, "root", EnvironmentRootPassword)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	for _, f := range c.Files {
		if err := env.vmDownloadFile(sshClient, f, filepath.Join(hostPath, f)); err != nil {
			return fmt.Errorf("cannot copy file %s: %s", f, err.Error())
		}
	}
	for _, d := range c.Dirs {
		if err := env.vmDownloadDir(sshClient, d, hostPath); err != nil {
			return err
		}
	}
	for src := range c.MapDirs {
		dir, pattern := filepath.Split(src)
		err := env.vmWalk(dir, func(path string, mode os.FileMode, linktarget string) error {
			envdir, filename := filepath.Split(path)
			match, _ := filepath.Match(pattern, filename)
			if match {
				var reldir string
				if envdir == "" {
					reldir = ""
				} else {
					reldir, err = filepath.Rel(dir, envdir)
					if err != nil {
						return err
					}
				}
				relpath := filepath.Join(dir, reldir, filename)
				destpath := filepath.Join(hostPath, relpath)
				if mode.IsDir() {
					return os.MkdirAll(destpath, 0755)
				}
				if (mode & os.ModeSymlink) == os.ModeSymlink {
					return os.Symlink(linktarget, destpath)
				}
				return env.vmDownloadFile(sshClient, path, destpath)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if c.Program != "" {
		err := env.vmDownloadFile(sshClient, c.Program, filepath.Join(hostPath, c.Program))
		if err != nil {
			return fmt.Errorf("cannot copy file %s: %s", c.Program, err.Error())
		}
		vmCmd := env.NewCommand("ldd " + c.Program).AsAdmin()
		vmCmd.SuppressOutput = true
		if err := vmCmd.Execute(); err != nil {
			return err
		}
		sharedLibs := vmCmd.StdOutput()
		for _, line := range strings.Split(sharedLibs, "\n") {
			lineFields := strings.Fields(line)
			for _, field := range lineFields {
				if field[0] == '/' {
					err := env.vmDownloadFile(sshClient, field, filepath.Join(hostPath, field))
					if err != nil {
						errString := fmt.Sprintf("cannot copy file %s: %s", field, err.Error())
						return errors.New(errString)
					}
					break
				}
			}
		}
	}
	return nil
}

// Shutdown stops the environment VM, if running.
func (env *Environment) Shutdown() error {
	vmCmd := env.NewCommand("shutdown", "-hP", "now").AsAdmin()
	err := vmCmd.Execute()
	env.PID = 0
	return err
}

// Uninstall uninstalls the environment from the system.
func (env *Environment) Uninstall() error {
	imageFilePath := filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension)
	return os.Remove(imageFilePath)
}

// NewEmptyCommand creates an empty command.
func (env *Environment) NewEmptyCommand() *EnvironmentCommand {
	return &EnvironmentCommand{
		Username: EnvironmentUsername,
		Password: EnvironmentUserPassword,
		Port:     env.SSHPort,
	}
}

// NewCommand creates new command to be executed via SSH.
func (env *Environment) NewCommand(command string, args ...interface{}) *EnvironmentCommand {
	vmCmd := env.NewEmptyCommand()
	vmCmd.CommandLines = []string{
		formatCommand(command, args...),
	}
	return vmCmd
}

// NewCommandf creates new formatted command to be executed via SSH.
func (env *Environment) NewCommandf(format string, args ...interface{}) *EnvironmentCommand {
	return env.NewCommand(fmt.Sprintf(format, args...))
}

// Downloads file from srcpath in VM to local filesystem.
func (env *Environment) vmDownloadFile(client *ssh.Client, srcpath, destpath string) error {
	log.Debugf("Copying file from remote %s to local %s", srcpath, destpath)
	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return err
	}
	defer scpClient.Close()
	if err := os.MkdirAll(filepath.Dir(destpath), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(destpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	return scpClient.CopyFromRemote(file, srcpath)
}

// Downloads a directory (recursively) from srcpath in VM to local filesystem.
func (env *Environment) vmDownloadDir(client *ssh.Client, srcpath, destpath string) error {
	log.Debugf("Copying directory from remote %s to local %s", srcpath, destpath)
	return env.vmWalk(srcpath, func(envpath string, mode os.FileMode, linktarget string) error {
		hostpath := filepath.Join(destpath, envpath)
		if mode.IsDir() {
			return os.MkdirAll(hostpath, 0755)
		}
		if (mode & os.ModeSymlink) == os.ModeSymlink {
			return os.Symlink(linktarget, hostpath)
		}
		return env.vmDownloadFile(client, envpath, hostpath)
	})
}

func (env *Environment) vmWalk(dir string, fn vmWalkFunc) error {
	vmCmd := env.NewCommand("ls -Al " + dir).AsAdmin()
	vmCmd.SuppressOutput = true
	if err := vmCmd.Execute(); err != nil {
		return err
	}
	if err := fn(dir, os.ModeDir, ""); err != nil {
		return err
	}
	dirContents := vmCmd.StdOutput()
	for _, line := range strings.Split(dirContents, "\n") {
		if line == "" {
			continue
		}
		var mode os.FileMode
		linktarget := ""
		lineFields := strings.Fields(line)
		nameEnd := len(lineFields)
		switch line[0] {
		case '-':
		case 'd':
			mode = os.ModeDir
		case 'l':
			mode = os.ModeSymlink
			for linkSep, field := range lineFields {
				if field == "->" {
					nameEnd = linkSep
					break
				}
			}
			linktarget = strings.Join(lineFields[nameEnd+1:], " ")
		default:
			continue
		}
		name := filepath.Join(dir, strings.Join(lineFields[8:nameEnd], " "))
		if mode == os.ModeDir {
			if err := env.vmWalk(name, fn); err != nil {
				return err
			}
		} else {
			if err := fn(name, mode, linktarget); err != nil {
				return err
			}
		}
	}
	return nil
}

type vmWalkFunc func(path string, mode os.FileMode, linktarget string) error

// Creates a new SSH client using the given port and credentials.
func newSSHClient(port int, username, password string) (*ssh.Client, error) {
	return ssh.Dial("tcp", fmt.Sprintf("localhost:%v", port), &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}
