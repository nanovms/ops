package crossbuild

import (
	"os"
	"path/filepath"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

var (
	// ErrMsgEnvironmentInitFailed occurs if environment initialization failed.
	ErrMsgEnvironmentInitFailed = "failed to initialize crossbuild environment"

	// Path to crossbuild home directory inside OPS home directory.
	crossBuildHomeDirPath = filepath.Join(lepton.GetOpsHome(), "crossbuild")
)

func init() {
	directories := []string{
		crossBuildHomeDirPath,
		vmImageDirPath,
	}

	for _, dir := range directories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				log.Fail(ErrMsgEnvironmentInitFailed, err)
			}
		}
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		conf := &Configuration{}
		if err := conf.Save(); err != nil {
			log.Fail(ErrMsgEnvironmentInitFailed, err)
		}
	}
}
