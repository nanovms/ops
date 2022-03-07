package cmd

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

	"github.com/spf13/pflag"
)

// NanosVersionCommandFlags is used to change configuration to use nanos version build tools paths
type NanosVersionCommandFlags struct {
	NanosVersion string
}

// MergeToConfig downloads specified nanos version build and change configuration nanos tools paths
func (flags *NanosVersionCommandFlags) MergeToConfig(config *types.Config) (err error) {
	nanosVersion := flags.NanosVersion

	if nanosVersion != "" {
		var exists bool
		exists, err = lepton.CheckNanosVersionExists(nanosVersion)
		if err != nil {
			return err
		}

		if !exists {
			err = lepton.DownloadReleaseImages(nanosVersion)
			if err != nil {
				return
			}
		}

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
