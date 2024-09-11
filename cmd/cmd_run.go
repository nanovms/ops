package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/onprem"
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

	persistentFlags.Bool("qmp", false, "qmp [local only]")

	return cmdRun
}

func runCommandHandler(cmd *cobra.Command, args []string) {

	c := lepton.NewConfig()

	program := args[0]
	c.Program = program
	var err error
	c.ProgramPath, err = filepath.Abs(c.Program)
	if err != nil {
		exitWithError(err.Error())
	}
	checkProgramExists(c.Program)

	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)

	buildImageFlags := NewBuildImageCommandFlags(flags)
	runLocalInstanceFlags := NewRunLocalInstanceCommandFlags(flags)

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, runLocalInstanceFlags)

	err = mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	qmp, _ := cmd.Flags().GetBool("qmp")
	if qmp {
		c.RunConfig.QMP = true
	}

	if c.Mounts != nil {
		err = onprem.AddVirtfsShares(c)
		if err != nil {
			exitWithError("Failed to add VirtFS shares: " + err.Error())
		}
	}

	if !runLocalInstanceFlags.SkipBuild {
		err = api.BuildImage(*c)
		if err != nil {
			fmt.Printf("Failed to build image, error is: %s", err)
			os.Exit(1)
		}
	}

	err = RunLocalInstance(c)
	if err != nil {
		exitWithError(err.Error())
	}
}
