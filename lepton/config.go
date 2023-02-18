package lepton

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// NewConfig construct instance of Config with default values
func NewConfig() *types.Config {
	c := new(types.Config)

	conf := os.Getenv("OPS_DEFAULT_CONFIG")
	if conf != "" {
		data, err := os.ReadFile(conf)
		if err != nil {
			log.Errorf("error reading config: %v\n", err)
		}
		err = json.Unmarshal(data, &c)
		if err != nil {
			log.Errorf("error config: %v\n", err)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return c
		}
		conf = filepath.Join(home, ".opsrc")

		if _, err = os.Stat(conf); err == nil {
			data, err := os.ReadFile(conf)
			if err != nil {
				log.Fatalf("error reading config: %v\n", err)
			}
			err = json.Unmarshal(data, &c)
			if err != nil {
				log.Fatalf("error config: %v\n", err)
			}
		}
	}

	c.RunConfig.Accel = true
	c.RunConfig.Memory = "2G"
	c.VolumesDir = LocalVolumeDir
	c.LocalFilesParentDirectory = "."

	return c
}
