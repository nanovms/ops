package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
)

// DeployCommand builds image and deploy an instance
func DeployCommand() *cobra.Command {
	var cmdDeploy = &cobra.Command{
		Use:   "deploy [ELF file]",
		Short: "Build an image from ELF and deploy an instance",
		Args:  cobra.MinimumNArgs(1),
		Run:   deployCommandHandler,
	}

	persistentFlags := cmdDeploy.PersistentFlags()

	PersistConfigCommandFlags(persistentFlags)
	PersistProviderCommandFlags(persistentFlags)
	PersistPkgCommandFlags(persistentFlags)
	PersistBuildImageCommandFlags(persistentFlags)
	PersistCreateInstanceFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)
	PersistNetConsoleFlags(persistentFlags)

	return cmdDeploy
}

func deployCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	createInstanceFlags := NewCreateInstanceCommandFlags(flags)
	netconsoleFlags := NewNetConsoleFlags(flags)

	c := lepton.NewConfig()

	program := args[0]
	c.Program = program
	var err error
	c.ProgramPath, err = filepath.Abs(c.Program)
	if err != nil {
		exitWithError(err.Error())
	}
	checkProgramExists(c.Program)

	if len(c.Args) == 0 {
		c.Args = []string{c.Program}
	} else {
		c.Args = append([]string{c.Program}, c.Args...)
	}

	mergeConfigContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, providerFlags, pkgFlags, createInstanceFlags, netconsoleFlags)
	err = mergeConfigContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
	if err != nil {
		exitWithError(err.Error())
	}

	// Delete image with the same name
	images, err := p.GetImages(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	for _, i := range images {
		if i.Name == ctx.Config().CloudConfig.ImageName {
			err = p.DeleteImage(ctx, ctx.Config().CloudConfig.ImageName)
			if err != nil {
				exitWithError(err.Error())
			}
		}
	}

	// Build image
	var keypath string
	if pkgFlags.Package != "" {
		keypath, err = p.BuildImageWithPackage(ctx, pkgFlags.PackagePath())
		if err != nil {
			exitWithError(err.Error())
		}
	} else {
		keypath, err = p.BuildImage(ctx)
		if err != nil {
			exitWithError("failed building image: " + err.Error())
		}
	}

	err = p.CreateImage(ctx, keypath)
	if err != nil {
		exitWithError(err.Error())
	}

	// Create instance and stop instances created with the same image
	ctx.Config().RunConfig.InstanceName = fmt.Sprintf("%v-%v",
		filepath.Base(c.CloudConfig.ImageName),
		strconv.FormatInt(time.Now().Unix(), 10),
	)

	ctx.Config().CloudConfig.Tags = append(ctx.Config().CloudConfig.Tags, types.Tag{Key: "image", Value: c.CloudConfig.ImageName})

	instances, err := p.GetInstances(ctx)
	if err != nil {
		exitWithError(err.Error())
	}

	err = p.CreateInstance(ctx)
	if err != nil {
		exitWithError("failed creating instance: " + err.Error())
	}

	for _, i := range instances {
		if i.Image == c.CloudConfig.ImageName {
			ctx.Logger().Debugf("deleting instance %s", i.Name)
			err := p.DeleteInstance(ctx, i.Name)
			if err != nil {
				exitWithError(err.Error())
			}
		}
	}
}
