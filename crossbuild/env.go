package crossbuild

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/xid"
)

const (
	// Environment home directory name.
	envDirName = ".opsenv"

	// Environment configuration file name.
	envConfigFileName = "environment.json"
)

// Environment is a crossbuild configuration.
type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	vm     *virtualMachine `json:"-"`
	source source          `json:"-"`
}

// Remove removes environment from local directory and VM.
func (env *Environment) Remove() error {
	if err := os.RemoveAll(envDirName); err != nil {
		return err
	}

	if err := env.vm.RemoveDir(env.vmDirPath()); err != nil {
		return err
	}
	return nil
}

// Resize changes VM disk size by given size.
func (env *Environment) Resize(size string) error {
	if err := env.vm.Resize(size); err != nil {
		return err
	}
	return nil
}

// Writes configuration to file.
func (env *Environment) Save() error {
	props := map[string]interface{}{
		envConfigFileName:    env,
		sourceConfigFileName: env.source,
	}

	for fileName, prop := range props {
		content, err := json.MarshalIndent(prop, "", "    ")
		if err != nil {
			return err
		}

		filePath := filepath.Join(envDirName, fileName)
		if err = ioutil.WriteFile(filePath, content, 0655); err != nil {
			return err
		}
	}
	return nil
}

// Sync synchronizes local environment with its copy in VM.
func (env *Environment) Sync() error {
	vmDirPath := env.vmDirPath()
	if err := env.vm.MkdirAll(vmDirPath); err != nil {
		return err
	}

	sshClient, err := newSSHClient(env.vm.ForwardPort, VMUsername, VMUserPassword)
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
		vmpath := filepath.Join(vmDirPath, relpath)

		if info.IsDir() {
			if err := env.vm.MkdirAll(vmpath); err != nil {
				return err
			}
			return nil
		}

		if err := env.vm.CopyFile(sshClient, path, vmpath, info.Mode()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Start starts environment.
func (env *Environment) Start() error {
	return env.vm.Start()
}

// Shutdown shuts down environment.
func (env *Environment) Shutdown() error {
	return env.vm.Shutdown()
}

// Running returns true if environment is runnning, otherwise false.
func (env *Environment) Running() bool {
	return env.vm.Alive()
}

// Exists returns true if environment image exists, otherwise false.
func (env *Environment) Exists() bool {
	if _, err := os.Stat(env.vm.ImageFilePath()); os.IsNotExist(err) {
		return false
	}
	return true
}

// Download pulls environment image from remote storage.
func (env *Environment) Download() error {
	return env.vm.Download()
}

// Returns path to environment directory inside VM.
func (env *Environment) vmDirPath() string {
	return filepath.Join("/home", VMUsername, "environments", env.ID)
}

// LoadEnvironment returns existing environment, or creates one if not exists.
func LoadEnvironment() (*Environment, error) {
	if err := os.MkdirAll(envDirName, 0755); err != nil {
		return nil, err
	}

	configFilePath := filepath.Join(envDirName, envConfigFileName)
	_, err := os.Stat(configFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	env := &Environment{}

	if err == nil {
		content, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(content, env); err != nil {
			return nil, err
		}

		// Load source config
		content, err = ioutil.ReadFile(filepath.Join(envDirName, sourceConfigFileName))
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(content, &env.source); err != nil {
			return nil, err
		}
	} else {
		env.ID = xid.New().String()
		env.Name = DefaultVMName
		env.source.Dependencies = make([]sourceDependency, 0)
		if err = env.Save(); err != nil {
			return nil, err
		}
	}

	vm, err := loadVM(env.Name)
	if err != nil {
		return nil, err
	}
	vm.workingDirPath = env.vmDirPath()
	env.vm = vm

	return env, nil
}
