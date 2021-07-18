package crossbuild

import (
	"strings"
)

// BuildUsingNexe creates executable using nexe (https://github.com/nexe/nexe).
func (env *Environment) BuildUsingNexe(filename string) error {
	filenameParts := strings.Split(filename, ".")
	executableName := filenameParts[0]

	vmCmd := env.NewCommandf(
		"cd %s && nexe -t 14.15.3 %s -o %s",
		env.vmSourcePath(),
		filename,
		executableName,
	).AsAdmin()
	if err := vmCmd.Execute(); err != nil {
		return err
	}

	return env.DownloadExecutableFile(executableName)
}
