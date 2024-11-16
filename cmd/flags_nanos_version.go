package cmd

import (
	"fmt"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

	"github.com/spf13/pflag"
)

// NanosVersionCommandFlags is used to change configuration to use nanos version build tools paths
type NanosVersionCommandFlags struct {
	NanosVersion string
}

func archPath() string {
	parch := ""
	if lepton.AltGOARCH != "" {
		if lepton.AltGOARCH == "arm64" {
			parch = "arm"
		}
	} else {
		if lepton.RealGOARCH == "arm64" {
			parch = "arm"
		}
	}

	return parch
}

// MergeToConfig downloads specified nanos version build and change configuration nanos tools paths
func (flags *NanosVersionCommandFlags) MergeToConfig(config *types.Config) (err error) {
	nanosVersion := flags.NanosVersion

	if nanosVersion != "" {
		arch := archPath()

		var exists bool

		exists, err = lepton.CheckNanosVersionExists(nanosVersion, arch)
		if err != nil {
			return err
		}

		if !exists {
			err = lepton.DownloadReleaseImages(nanosVersion, arch)
			if err != nil {
				return
			}
		}

		// override Boot and Kernel parameters in configuration file
		config.Boot = ""
		config.Kernel = ""
		updateNanosToolsPaths(config, nanosVersion)
		config.NanosVersion = nanosVersion
	} else {
		config.NanosVersion = lepton.LocalReleaseVersion
	}

	return
}

// NewNanosVersionCommandFlags returns an instance of NanosVersionCommandFlags
func NewNanosVersionCommandFlags(cmdFlags *pflag.FlagSet) (flags *NanosVersionCommandFlags) {
	var err error
	flags = &NanosVersionCommandFlags{}

	flags.NanosVersion, err = cmdFlags.GetString("nanos-version")
	if err != nil {
		exitWithError(err.Error())
	}

	return
}

// PersistNanosVersionCommandFlags append nanos version flag to a command
func PersistNanosVersionCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.StringP("nanos-version", "", "", "uses nanos tools version")
}
