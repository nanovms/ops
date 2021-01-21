package cmd

import (
	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

// LoadCommand helps you to run application with package
func LoadCommand() *cobra.Command {
	var cmdLoadPackage = &cobra.Command{
		Use:   "load [packagename]",
		Short: "load and run a package from ['ops pkg list']",
		Args:  cobra.MinimumNArgs(1),
		Run:   loadCommandHandler,
	}

	PersistConfigCommandFlags(cmdLoadPackage.PersistentFlags())
	PersistBuildImageCommandFlags(cmdLoadPackage.PersistentFlags())
	PersistStartImageCommandFlags(cmdLoadPackage.PersistentFlags())
	cmdLoadPackage.PersistentFlags().BoolP("local", "l", false, "load local package")

	return cmdLoadPackage
}

func loadCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(cmd.Flags())
	buildImageFlags := NewBuildImageCommandFlags(flags)
	startImageFlags := NewStartImageCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	pkgFlags.Package = args[0]

	c := lepton.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, buildImageFlags, globalFlags, startImageFlags, pkgFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	if !startImageFlags.SkipBuild {
		setNanosBaseImage(c)
		if err = api.BuildImageFromPackage(pkgFlags.PackagePath(), *c); err != nil {
			panic(err)
		}
	}

	err = StartUnikernel(c)
	if err != nil {
		exitWithError(err.Error())
	}
}
