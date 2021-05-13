package cmd

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/spf13/cobra"
)

// BuildCommand helps you to build image from ELF
func BuildCommand() *cobra.Command {
	var cmdBuild = &cobra.Command{
		Use:   "build [ELF file]",
		Short: "Build an image from ELF",
		Args:  cobra.MinimumNArgs(1),
		Run:   buildCommandHandler,
	}

	persistentFlags := cmdBuild.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistProviderCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)

	return cmdBuild
}

func buildCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)

	c := lepton.NewConfig()

	c.Program = args[0]
	checkProgramExists(c.Program)

	mergeConfigContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags)
	err := mergeConfigContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	providerFlags := NewProviderCommandFlags(flags)

	p, ctx, err := getProviderAndContext(c, providerFlags.TargetCloud)
	if err != nil {
		exitWithError(err.Error())
	}

	var imagePath string

	if imagePath, err = p.BuildImage(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Bootable image file:%s\n", imagePath)
}
