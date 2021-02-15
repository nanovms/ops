package cmd

import (
	"os"
	"path"

	"github.com/nanovms/ops/config"
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

	PersistConfigCommandFlags(cmdRun.PersistentFlags())
	PersistBuildImageCommandFlags(cmdRun.PersistentFlags())
	PersistStartImageCommandFlags(cmdRun.PersistentFlags())

	return cmdRun
}

func runCommandHandler(cmd *cobra.Command, args []string) {

	c := config.NewConfig()

	program := args[0]
	c.Program = program
	curdir, _ := os.Getwd()
	c.ProgramPath = path.Join(curdir, c.Program)
	checkProgramExists(c.Program)

	if len(c.Args) == 0 {
		c.Args = []string{c.Program}
	} else {
		c.Args = append([]string{c.Program}, c.Args...)
	}

	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	startImageFlags := NewStartImageCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, buildImageFlags, startImageFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	if !startImageFlags.SkipBuild {
		err = api.BuildImage(*c)
		if err != nil {
			panic(err)
		}
	}

	err = StartUnikernel(c)
	if err != nil {
		exitWithError(err.Error())
	}
}
