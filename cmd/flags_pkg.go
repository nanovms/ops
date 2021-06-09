package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
)

// PkgCommandFlags consolidates all command flags required to use a provider
type PkgCommandFlags struct {
	Package      string
	LocalPackage bool
}

// PackagePath returns the package path in file system
func (flags *PkgCommandFlags) PackagePath() string {
	if flags.LocalPackage {
		return path.Join(api.GetOpsHome(), "local_packages", flags.Package)
	}

	return path.Join(api.GetOpsHome(), "packages", flags.Package)
}

// MergeToConfig merge package configuration to ops configuration
func (flags *PkgCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if flags.Package == "" {
		return
	}

	packagePath := flags.PackagePath()

	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		if flags.LocalPackage {
			return fmt.Errorf("No local package with the name %s found", flags.Package)
		}

		downloadPackage(flags.Package, c)
	}

	manifestPath := path.Join(packagePath, "package.manifest")
	if _, err := os.Stat(manifestPath); err != nil {
		return errors.New("failed finding package manifest")
	}

	pkgConfig := &types.Config{}
	err = unWarpConfig(manifestPath, pkgConfig)
	if err != nil {
		return err
	}

	c.Program = pkgConfig.Program
	c.Version = pkgConfig.Version

	c.Language = pkgConfig.Language
	c.Runtime = pkgConfig.Runtime
	c.Description = pkgConfig.Description

	c.Args = append(pkgConfig.Args, c.Args...)
	c.Dirs = append(pkgConfig.Dirs, c.Dirs...)
	c.Files = append(pkgConfig.Files, c.Files...)

	if c.MapDirs == nil {
		c.MapDirs = make(map[string]string)
	}

	if c.Env == nil {
		c.Env = make(map[string]string)
	}

	for k, v := range pkgConfig.MapDirs {
		c.MapDirs[k] = v
	}

	for k, v := range pkgConfig.Env {
		c.Env[k] = v
	}

	if c.BaseVolumeSz == "" {
		c.BaseVolumeSz = pkgConfig.BaseVolumeSz
	}

	if c.NameServer == "" {
		c.NameServer = pkgConfig.NameServer
	}

	if c.TargetRoot == "" {
		c.TargetRoot = pkgConfig.TargetRoot
	}

	imageName := c.RunConfig.Imagename
	images := path.Join(lepton.GetOpsHome(), "images")
	if imageName == "" {
		c.RunConfig.Imagename = path.Join(images, filepath.Base(pkgConfig.Program))
		c.CloudConfig.ImageName = fmt.Sprintf("%v-image", filepath.Base(pkgConfig.Program))
	} else if c.CloudConfig.ImageName == "" {
		imageName = path.Join(images, filepath.Base(imageName))
		c.CloudConfig.ImageName = imageName
	}

	return
}

// NewPkgCommandFlags returns an instance of PkgCommandFlags
func NewPkgCommandFlags(cmdFlags *pflag.FlagSet) (flags *PkgCommandFlags) {
	var err error
	flags = &PkgCommandFlags{}

	flags.LocalPackage, err = cmdFlags.GetBool("local")
	if err != nil {
		exitWithError(err.Error())
	}

	// error handling is ignored because load command reads package from argument
	flags.Package, _ = cmdFlags.GetString("package")

	return
}

// PersistPkgCommandFlags append a command the required flags to use a package
func PersistPkgCommandFlags(cmdFlags *pflag.FlagSet) {
	cmdFlags.BoolP("local", "l", false, "load local package")
	cmdFlags.String("package", "", "ops package name")
}
