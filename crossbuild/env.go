package crossbuild

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	// EnvironmentDownloadBaseURL is the base url to download environment image.
	EnvironmentDownloadBaseURL = "storage.googleapis.com/nanos-build-envs"

	// SupportedEnvironmentsFileName is name of the file that list supported environments.
	SupportedEnvironmentsFileName = "environments.json"

	// EnvironmentImageFileExtension is extension for environment image file.
	EnvironmentImageFileExtension = ".qcow2"

	// EnvironmentPIDFileExtension is extension for environment process id file.
	EnvironmentPIDFileExtension = ".pid"

	// EnvironmentUsername is username for regular account.
	EnvironmentUsername = "user"

	// EnvironmentUserPassword is password for regular account.
	EnvironmentUserPassword = "user"

	// EnvironmentRootPassword is password for admin account.
	EnvironmentRootPassword = "root"
)

var (
	// EnvironmentImageDirPath is path to directory that keeps environment images.
	EnvironmentImageDirPath = filepath.Join(CrossBuildHomeDirPath, "images")
)

// Environment is a virtualized crossbuild operating system.
type Environment struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Version          string             `json:"version"`
	IsDefault        bool               `json:"is_default"`
	Config           *EnvironmentConfig `json:"-"`
	PID              int                `json:"-"`
	WorkingDirectory string             `json:"-"`
}

// Download pulls environment images from remote storage.
func (env *Environment) Download() error {
	compressedImgFileName := env.ID + ".zip"
	compressedImgFilePath := filepath.Join(EnvironmentImageDirPath, compressedImgFileName)
	downloadURL := "https://" + path.Join(EnvironmentDownloadBaseURL, compressedImgFileName)
	if err := lepton.DownloadFile(compressedImgFilePath, downloadURL, 600, true); err != nil {
		return err
	}

	if err := util.Unzip(compressedImgFilePath, EnvironmentImageDirPath); err != nil {
		return err
	}

	if err := os.Remove(compressedImgFilePath); err != nil {
		log.Warn(err)
	}
	return nil
}

// Boot starts environment VM.
func (env *Environment) Boot() error {
	if env.PID > 0 {
		return nil // Environment is running
	}

	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		return errors.New("no hypervisor found in PATH")
	}

	cmd := hypervisor.Command(&types.RunConfig{
		Accel:     true,
		Imagename: filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension),
		Memory:    env.Config.Memory,
		Ports: []string{
			fmt.Sprintf("%d-%d", env.Config.Port, 22),
		},
	})

	if err := cmd.Start(); err != nil {
		return err
	}

	errBootFailure := errors.New("environment boot failure")
	if cmd.Process == nil {
		return errBootFailure
	}
	env.PID = cmd.Process.Pid

	var (
		wg         = &sync.WaitGroup{}
		timedOut   = false
		checkCount = 0

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
			time.Sleep(time.Second)

			if !keepChecking {
				break
			}

			if env.IsReady() {
				break
			}

			checkCount++
			if checkCount >= 60 {
				timedOut = true
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()

	if interrupted {
		return errBootFailure
	}

	if timedOut {
		return errors.New("environment is not responding")
	}

	return nil
}

// Shutdown stops this environment, if running.
func (env *Environment) Shutdown() error {
	vmCmd := env.NewCommand("shutdown", "-hP", "now").AsAdmin()
	return vmCmd.Execute()
}

// IsReady returns true if environment can establish communication.
func (env *Environment) IsReady() bool {
	vmCmd := env.NewCommand("ls /")
	vmCmd.SupressOutput = true
	if err := vmCmd.Execute(); err != nil {
		log.Warn(err)
		return false
	}
	return vmCmd.CombinedOutput() != ""
}

// Sync synchronizes local environment with its copy in VM.
func (env *Environment) Sync() error {
	vmSourcePath := env.vmSourcePath()
	if err := env.vmMkdirAll(vmSourcePath); err != nil {
		return err
	}

	sshClient, err := newSSHClient(env.Config.Port, EnvironmentUsername, EnvironmentUserPassword)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := filepath.Walk(currentDir, func(path string, info fs.FileInfo, err error) error {
		relpath := strings.TrimPrefix(path, currentDir)
		vmpath := filepath.Join(vmSourcePath, relpath)

		if info.IsDir() {
			if filepath.Base(path) == "bin" {
				return nil
			}

			if err := env.vmMkdirAll(vmpath); err != nil {
				return err
			}
			return nil
		}

		parentDir := filepath.Base(filepath.Dir(path))
		if parentDir == "bin" {
			return nil
		}

		if err := env.vmCopyFile(sshClient, path, vmpath, info.Mode()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Resize changes disk size by given size.
// The value of size follows the rule of 'qemu-img resize' input.
func (env *Environment) Resize(size string) error {
	imgFilePath := filepath.Join(EnvironmentImageDirPath, env.ID+EnvironmentImageFileExtension)
	resizedImgFilePath := filepath.Join(EnvironmentImageDirPath, env.ID+"-resized"+EnvironmentImageFileExtension)

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

	isShrink := strings.HasPrefix(size, "-")

	args := []string{"resize"}
	if isShrink {
		args = append(args, "--shrink")
	}
	args = append(args, imgFilePath, size)
	cmd := exec.Command("qemu-img", args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		// log.Errorf("%s", string(output))
		return err
	}

	op := "--expand"
	if isShrink {
		op = "--shrink"
	}
	cmd = exec.Command("sudo", "virt-resize", op, "/dev/sda1", imgFilePath, resizedImgFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		// log.Errorf("%s", string(output))
		return err
	}
	// log.Info(string(output))

	if err := os.Remove(imgFilePath); err != nil {
		return err
	}

	if err := os.Rename(resizedImgFilePath, imgFilePath); err != nil {
		return err
	}
	return nil
}

// DefaultEnvironment returns default environment.
func DefaultEnvironment() (*Environment, error) {
	config, err := LoadConfiguration()
	if err != nil {
		return nil, err
	}

	if config.DefaultEnvironment != "" {
		return LoadEnvironment(config.DefaultEnvironment)
	}

	environments, err := SupportedEnvironments()
	if err != nil {
		return nil, err
	}
	envDefault := environments.Default()
	return LoadEnvironment(envDefault.ID)
}

// LoadEnvironment loads environment identified with given id.
func LoadEnvironment(id string) (*Environment, error) {
	environments, err := SupportedEnvironments()
	if err != nil {
		return nil, err
	}

	env := environments.Find(id)
	if env == nil {
		return nil, errors.New("requested environment is not supported")
	}

	config, err := LoadConfiguration()
	if err != nil {
		return nil, err
	}
	envConfig := config.FindEnvironment(id)
	if envConfig == nil {
		envConfig = &EnvironmentConfig{
			ID:     env.ID,
			Port:   config.newForwardPort(),
			Memory: "2G",
		}

		config.Environments = append(config.Environments, *envConfig)
		if err = config.Save(); err != nil {
			return nil, err
		}
	}
	env.Config = envConfig

	imageFilePath := filepath.Join(EnvironmentImageDirPath, envConfig.ID+".qcow2")
	if _, err = os.Stat(imageFilePath); os.IsNotExist(err) {
		fmt.Println("Download environment: ", envConfig.ID)
		if err = env.Download(); err != nil {
			return nil, err
		}
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	env.WorkingDirectory = strings.TrimPrefix(currentDir, userHomeDir)

	pid, err := environmentProcessID(env)
	if err != nil {
		return nil, err
	}
	if pid == 0 {
		return env, nil
	}
	env.PID = pid

	if !env.IsReady() {
		pidStr := strconv.Itoa(env.PID)
		cmd := exec.Command("kill", "-9", pidStr)
		if err := cmd.Run(); err != nil {
			return nil, errors.New("environment is not responding and cannot be killed, please do it manually for PID " + pidStr)
		}
		log.Warn("environment is not responding and succesfully killed")
	}
	return env, nil
}
