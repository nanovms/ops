package crossbuild

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"
	"github.com/nanovms/ops/util"
	"golang.org/x/crypto/ssh"
)

const (
	// DefaultVMName is the default VM to be used.
	DefaultVMName = "debian-10.10.0-amd64"

	// VMUsername is VM's login username.
	VMUsername = "user"

	// VMUserPassword is VM's login password.
	VMUserPassword = "user"

	// VMRootPassword is VM's root login password.
	VMRootPassword = "root"

	// VMImageDownloadBaseURL is the base url to download a VM image.
	VMImageDownloadBaseURL = "storage.googleapis.com/nanos-build-envs"

	// VMImageFileExtension is extension for VM image file.
	VMImageFileExtension = ".qcow2"
)

var (
	// Path to directory that stores VM images.
	VMImageDirPath = filepath.Join(CrossBuildHomeDirPath, "vm_images")
)

// VM is a virtualized OS for crossbuild execution.
type VM struct {
	Name           string `json:"name"`
	ForwardPort    int    `json:"forward_port"`
	PID            int    `json:"-"`
	MemorySize     string `json:"memory_size"`
	WorkingDirPath string `json:"-"`
}

// ImageFilePath returns path to VM image file.
func (vm *VM) ImageFilePath() string {
	return filepath.Join(VMImageDirPath, vm.Name+VMImageFileExtension)
}

// ImagePIDFilePath returns path to VM PID file.
func (vm *VM) ImagePIDFilePath() string {
	return filepath.Join(VMImageDirPath, vm.Name+".pid")
}

// Download VM image, if not exists.
func (vm *VM) Download() error {
	compressedImgFileName := vm.Name + ".zip"
	compressedImgFilePath := filepath.Join(VMImageDirPath, compressedImgFileName)
	downloadURL := "https://" + path.Join(VMImageDownloadBaseURL, compressedImgFileName)
	if err := lepton.DownloadFile(compressedImgFilePath, downloadURL, 600, true); err != nil {
		return err
	}

	if err := util.Unzip(compressedImgFilePath, VMImageDirPath); err != nil {
		return err
	}

	os.Remove(compressedImgFilePath)
	return nil
}

// Alive returns true if VM is running and can establish communication.
func (vm *VM) Alive() bool {
	if vm.PID > 0 {
		vmCmd := vm.NewCommand("ls /")
		vmCmd.SupressOutput = true
		if err := vmCmd.Execute(); err != nil {
			log.Warn(err)
			return false
		}

		return vmCmd.CombinedOutput() != ""
	}
	return false
}

// Starts this virtual machine.
// If the image does not exists, it will be downloaded.
func (vm *VM) Start() error {
	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		return errors.New("no hypervisor found in PATH")
	}

	cmd := hypervisor.Command(&types.RunConfig{
		Accel:     true,
		Imagename: vm.ImageFilePath(),
		Memory:    vm.MemorySize,
		Ports: []string{
			fmt.Sprintf("%d-%d", vm.ForwardPort, 22),
		},
	})

	if err := cmd.Start(); err != nil {
		return err
	}

	if cmd.Process == nil {
		return errors.New("VM fail to start")
	}

	vm.PID = cmd.Process.Pid
	pid := strconv.Itoa(cmd.Process.Pid)
	if err := ioutil.WriteFile(vm.ImagePIDFilePath(), []byte(pid), 0655); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			log.Warn(err)
		}
		return err
	}

	var (
		wg           = &sync.WaitGroup{}
		timedOut     = false
		chError      error
		processAlive = false

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

	go func() {
		for {
			time.Sleep(time.Millisecond * 100)

			if !keepChecking {
				break
			}

			if !processAlive {
				if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
					chError = err
					break
				}
				processAlive = true
			}

			if vm.Alive() {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()

	if interrupted {
		return errors.New("VM initialization interrupted")
	}

	if chError != nil {
		return chError
	}

	if timedOut {
		return errors.New("no response from VM, probably it is not started or the ssh server is not running")
	}

	return nil
}

// Shutdown stops this VM, if running.
func (vm *VM) Shutdown() error {
	vmCmd := vm.NewCommand("shutdown", "-hP", "now").AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			alive, err := isProcessAlive(vm.PID)
			if err != nil {
				log.Error(err)
				break
			}

			if !alive {
				break
			}
			time.Sleep(time.Second)
		}
		wg.Done()
	}()
	wg.Wait()

	os.Remove(vm.ImagePIDFilePath())
	return nil
}

// Resize changes disk size by given size.
// The value of size follows the rule of 'qemu-img resize' input.
func (vm *VM) Resize(size string) error {
	imgFilePath := vm.ImageFilePath()
	imgFilePathWithoutExt := strings.TrimSuffix(imgFilePath, VMImageFileExtension)
	resizedImgFilePath := imgFilePathWithoutExt + "-resized" + VMImageFileExtension

	defer os.Remove(resizedImgFilePath)

	_, err := os.Stat(resizedImgFilePath)
	if err == nil {
		if err = os.Remove(resizedImgFilePath); err != nil {
			return err
		}
	}

	imgFile, err := os.Open(imgFilePath)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	resizedImgFile, err := os.Create(resizedImgFilePath)
	if err != nil {
		return err
	}
	defer resizedImgFile.Close()

	if _, err := io.Copy(resizedImgFile, imgFile); err != nil {
		return err
	}

	resizedImgFile.Close()
	imgFile.Close()

	cmd := exec.Command("qemu-img", "resize", imgFilePath, size)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("%s", string(output))
		return err
	}
	log.Info(string(output))

	cmd = exec.Command("sudo", "virt-resize", "--expand", "/dev/sda1", imgFilePath, resizedImgFilePath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Errorf("%s", string(output))
		return err
	}
	log.Info(string(output))

	if err := os.Remove(imgFilePath); err != nil {
		return err
	}

	if err := os.Rename(resizedImgFilePath, imgFilePath); err != nil {
		return err
	}
	return nil
}

// MkdirAll executes 'mkdir -p' command in VM.
func (vm *VM) MkdirAll(dirpath string) error {
	vmCmd := vm.NewCommand("mkdir", "-p", dirpath)
	return vmCmd.Execute()
}

// RemoveDir removed directory recursively given its path.
func (vm *VM) RemoveDir(dirpath string) error {
	vmCmd := vm.NewCommand("rm", "-rf", dirpath)
	return vmCmd.Execute()
}

// CopyFile copies file from srcpath in local filesystem to destpath in VM.
func (vm *VM) CopyFile(client *ssh.Client, srcpath, destpath string, permissions fs.FileMode) error {
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

// NewCommand creates new command to be executed using user account.
func (vm *VM) NewCommand(command string, args ...interface{}) *VMCommand {
	return &VMCommand{
		CommandLines: []string{
			formatCommand(command, args...),
		},
		Username: VMUsername,
		Password: VMUserPassword,
		Port:     vm.ForwardPort,
	}
}

// NewCommandf creates new formatted command to be executed using user account.
func (vm *VM) NewCommandf(format string, args ...interface{}) *VMCommand {
	return &VMCommand{
		CommandLines: []string{
			fmt.Sprintf(format, args...),
		},
		Username: VMUsername,
		Password: VMUserPassword,
		Port:     vm.ForwardPort,
	}
}

// Loads existing VM configuration, or creates one if doesn't exist.
func loadVM(name string) (*VM, error) {
	config, err := LoadConfiguration()
	if err != nil {
		return nil, err
	}

	var vm *VM
	for _, v := range config.VirtualMachines {
		if v.Name == name {
			vm = v
			break
		}
	}

	if vm == nil {
		vm = &VM{
			Name:        name,
			ForwardPort: config.newForwardPort(),
		}
		config.VirtualMachines = append(config.VirtualMachines, vm)
		if err := config.Save(); err != nil {
			return nil, err
		}
	}

	if vm.MemorySize == "" {
		vm.MemorySize = "2G"
	}

	// Check running process.
	pidFilePath := vm.ImagePIDFilePath()
	_, err = os.Stat(pidFilePath)
	if os.IsNotExist(err) {
		return vm, nil
	}
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return nil, err
	}
	pidStr := strings.Trim(string(content), " ")
	pidStr = strings.TrimSuffix(pidStr, "\n")
	if pidStr == "" {
		os.Remove(pidFilePath)
		return vm, err
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, err
	}

	alive, err := isProcessAlive(pid)
	if err != nil {
		return nil, err
	}

	if alive {
		vm.PID = pid
	} else {
		os.Remove(pidFilePath)
	}
	return vm, nil
}

// VMCommand is one or more command lines to be executed inside VM.
type VMCommand struct {
	SupressOutput bool
	CommandLines  []string
	Username      string
	Password      string
	Port          int
	OutputBuffer  bytes.Buffer
	ErrorBuffer   bytes.Buffer
}

// CombinedOutput returns combined string of standard output and error.
func (cmd *VMCommand) CombinedOutput() string {
	return cmd.OutputBuffer.String() + cmd.ErrorBuffer.String()
}

// Execute executes command in VM as regular user, or as admin if given asAdmin is true.
func (cmd *VMCommand) Execute() error {
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
func (cmd *VMCommand) Then(command string, args ...interface{}) *VMCommand {
	cmd.CommandLines = append(cmd.CommandLines, formatCommand(command, args...))
	return cmd
}

// AsAdmin modified this command to use admin account.
func (cmd *VMCommand) AsAdmin() *VMCommand {
	cmd.Username = "root"
	cmd.Password = VMRootPassword
	return cmd
}

// Executes a single command.
func (cmd *VMCommand) executeCommand(client *ssh.Client, cmdLine string) error {
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

// Formats given command and arguments as a single string.
func formatCommand(command string, args ...interface{}) string {
	line := []interface{}{command}
	line = append(line, args...)
	return fmt.Sprintln(line...)
}
