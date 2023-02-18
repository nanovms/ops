package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

	"github.com/spf13/pflag"
)

// PkgCommandFlags consolidates all command flags required to use a provider
type PkgCommandFlags struct {
	Package        string
	SluggedPackage string
	LocalPackage   bool
}

// PackagePath returns the package path in file system
func (flags *PkgCommandFlags) PackagePath() string {
	if flags.LocalPackage {
		return path.Join(api.GetOpsHome(), "local_packages", flags.Package)
	}
	flags.SluggedPackage = strings.ReplaceAll(flags.Package, ":", "_")

	// if the local_path doesn't exist then we try to check if there is a "v" version
	pkgPath := path.Join(api.GetOpsHome(), "packages", flags.SluggedPackage)
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		pkgPath = flags.buildAlternatePath()
	}

	return pkgPath
}

// buildAlternatePath generates an alternate package path with an extra "v" prefixed
// to the version. There are certain packages on pkghub which follow these conventions
func (flags *PkgCommandFlags) buildAlternatePath() string {
	pkgParts := strings.Split(flags.Package, ":")
	pkgPath := path.Join(api.GetOpsHome(), "packages", flags.SluggedPackage)
	if len(pkgParts) > 1 {
		pkg := pkgParts[0]
		version := "v" + pkgParts[1]
		fuzzyPkgVersion := pkg + "_" + version
		altPkgPath := path.Join(api.GetOpsHome(), "packages", fuzzyPkgVersion)
		if _, err := os.Stat(altPkgPath); os.IsNotExist(err) {
			return pkgPath
		}
		flags.SluggedPackage = fuzzyPkgVersion
		return altPkgPath
	}

	return pkgPath
}

// MergeToConfig merge package configuration to ops configuration
func (flags *PkgCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if flags.Package == "" {
		return
	}

	packagePath := flags.PackagePath()
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		if flags.LocalPackage {
			return fmt.Errorf("no local package with the name %s found", flags.Package)
		}

		downloadPackage(flags.Package, c)
	}

	// re-evaluate the package path to make sure correct paths are detected
	packagePath = flags.PackagePath()
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

	if len(c.NameServers) == 0 {
		c.NameServers = pkgConfig.NameServers
	}

	if c.TargetRoot == "" {
		c.TargetRoot = pkgConfig.TargetRoot
	}

	imageName := c.RunConfig.ImageName
	images := path.Join(lepton.GetOpsHome(), "images")
	if imageName == "" {
		c.RunConfig.ImageName = path.Join(images, filepath.Base(pkgConfig.Program))
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
