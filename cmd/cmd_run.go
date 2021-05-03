package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

// RunCommand provides support for running binary with nanos
func RunCommand() *cobra.Command {
	var cmdRun = &cobra.Command{
		Use:   "run [elf]",
		Short: "Run ELF binary as unikernel",
		Args:  cobra.MinimumNArgs(1),
		Run:   runCommandHandler,
	}

	persistentFlags := cmdRun.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistRunLocalInstanceCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	return cmdRun
}

func runCommandHandler(cmd *cobra.Command, args []string) {

	c := lepton.NewConfig()

	program := args[0]
	c.Program = program
	curdir, _ := os.Getwd()
	c.ProgramPath = path.Join(curdir, c.Program)
	checkProgramExists(c.Program)

	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	runLocalInstanceFlags := NewRunLocalInstanceCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, runLocalInstanceFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	if !runLocalInstanceFlags.SkipBuild {
		err = api.BuildImage(*c)
		if err != nil {
			panic(err)
		}
	}

	err = RunLocalInstance(c)
	if err != nil {
		exitWithError(err.Error())
	}
}
