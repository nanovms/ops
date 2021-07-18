package crossbuild

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

var (
	// ConfigFilePath is path to crossbuild configuration file.
	ConfigFilePath = filepath.Join(CrossBuildHomeDirPath, "config.json")
)

// Configuration is configurable crossbuild settings.
type Configuration struct {
	DefaultEnvironment string              `json:"default_environment"`
	Environments       []EnvironmentConfig `json:"environments"`
	usedVMPorts        map[int]bool        `json:"-"`
}

// Save writes configuration to file located at OPS_HOME_DIR/crossbuild/config.json.
func (cfg *Configuration) Save() error {
	content, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(ConfigFilePath, content, 0655); err != nil {
		return err
	}
	return nil
}

// FindEnvironment returns environment configuration identified by given id, if any.
func (cfg *Configuration) FindEnvironment(id string) *EnvironmentConfig {
	if len(cfg.Environments) == 0 {
		return nil
	}

	for _, env := range cfg.Environments {
		if env.ID == id {
			return &env
		}
	}
	return nil
}

// LoadConfiguration reads configuration file located at OPS_HOME_DIR/crossbuild/config.json.
func LoadConfiguration() (*Configuration, error) {
	content, err := ioutil.ReadFile(ConfigFilePath)
	if err != nil {
		return nil, err
	}

	cfg := &Configuration{}
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, err
	}

	usedPorts := make(map[int]bool)
	vmCount := len(cfg.Environments)
	for i := 0; i < vmCount; i++ {
		usedPorts[cfg.Environments[i].Port] = true
	}
	cfg.usedVMPorts = usedPorts
	return cfg, nil
}

func (cfg *Configuration) newForwardPort() int {
	port := 20000
	for {
		if _, used := cfg.usedVMPorts[port]; !used {
			break
		}
		port++
	}
	return port
}
