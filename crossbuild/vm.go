package crossbuild

import (
	"errors"
	"fmt"
	"io"
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

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/nanovms/ops/types"
	"github.com/nanovms/ops/util"
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
	vmImageDirPath = filepath.Join(crossBuildHomeDirPath, "vm_images")
)

// Virtualized OS for crossbuild execution, a.k.a VM.
type virtualMachine struct {
	Name        string `json:"name"`
	ForwardPort int    `json:"forward_port"`
	PID         int    `json:"-"`
	MemorySize  string `json:"memory_size"`

	workingDirPath string `json:"-"`
}

// ImageFilePath returns path to VM image file.
func (vm *virtualMachine) ImageFilePath() string {
	return filepath.Join(vmImageDirPath, vm.Name+VMImageFileExtension)
}

// ImagePIDFilePath returns path to VM PID file.
func (vm *virtualMachine) ImagePIDFilePath() string {
	return filepath.Join(vmImageDirPath, vm.Name+".pid")
}

// Download VM image, if not exists.
func (vm *virtualMachine) Download() error {
	compressedImgFileName := vm.Name + ".zip"
	compressedImgFilePath := filepath.Join(vmImageDirPath, compressedImgFileName)
	downloadURL := "https://" + path.Join(VMImageDownloadBaseURL, compressedImgFileName)
	if err := lepton.DownloadFile(compressedImgFilePath, downloadURL, 600, true); err != nil {
		return err
	}

	if err := util.Unzip(compressedImgFilePath, vmImageDirPath); err != nil {
		return err
	}

	os.Remove(compressedImgFilePath)
	return nil
}

// Alive returns true if VM is running and can establish communication.
func (vm *virtualMachine) Alive() bool {
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
func (vm *virtualMachine) Start() error {
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
func (vm *virtualMachine) Shutdown() error {
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
func (vm *virtualMachine) Resize(size string) error {
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

// Loads existing VM configuration, or creates one if doesn't exist.
func loadVM(name string) (*virtualMachine, error) {
	config, err := LoadConfiguration()
	if err != nil {
		return nil, err
	}

	var vm *virtualMachine
	for _, v := range config.VirtualMachines {
		if v.Name == name {
			vm = v
			break
		}
	}

	if vm == nil {
		vm = &virtualMachine{
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
