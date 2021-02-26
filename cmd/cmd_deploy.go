package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

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

	PersistConfigCommandFlags(cmdDeploy.PersistentFlags())
	PersistProviderCommandFlags(cmdDeploy.PersistentFlags())
	PersistPkgCommandFlags(cmdDeploy.PersistentFlags())
	PersistBuildImageCommandFlags(cmdDeploy.PersistentFlags())
	PersistCreateInstanceFlags(cmdDeploy.PersistentFlags())
	PersistNightlyCommandFlags(cmdDeploy.PersistentFlags())

	return cmdDeploy
}

func deployCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	createInstanceFlags := NewCreateInstanceCommandFlags(flags)

	c := types.NewConfig()

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

	mergeConfigContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, buildImageFlags, providerFlags, pkgFlags, createInstanceFlags)
	err := mergeConfigContainer.Merge(c)
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
			ctx.Logger().Debug("deleting instance %s", i.Name)
			err := p.DeleteInstance(ctx, i.Name)
			if err != nil {
				exitWithError(err.Error())
			}
		}
	}
}
