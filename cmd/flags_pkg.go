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
func (flags *PkgCommandFlags) PackagePath() (packagePath string) {
	if flags.LocalPackage {
		packagePath = path.Join(api.GetOpsHome(), "local_packages", flags.Package)
	} else {
		packagePath = downloadAndExtractPackage(flags.Package)
	}
	return
}

// MergeToConfig merge package configuration to ops configuration
func (flags *PkgCommandFlags) MergeToConfig(c *types.Config) (err error) {
	if flags.Package == "" {
		return
	}

	packagePath := flags.PackagePath()

	manifestPath := path.Join(packagePath, "package.manifest")
	if _, err := os.Stat(manifestPath); err != nil {
		return errors.New("failed finding package manifest")
	}

	pkgConfig := unWarpConfig(manifestPath)

	pkgConfig.Args = append(pkgConfig.Args, c.Args...)
	pkgConfig.Dirs = append(pkgConfig.Dirs, c.Dirs...)
	pkgConfig.Files = append(pkgConfig.Files, c.Files...)

	if pkgConfig.MapDirs == nil {
		pkgConfig.MapDirs = make(map[string]string)
	}

	if pkgConfig.Env == nil {
		pkgConfig.Env = make(map[string]string)
	}

	for k, v := range c.MapDirs {
		pkgConfig.MapDirs[k] = v
	}

	for k, v := range c.Env {
		pkgConfig.Env[k] = v
	}

	if c.BaseVolumeSz != "" {
		pkgConfig.BaseVolumeSz = c.BaseVolumeSz
	}

	if c.NameServer != "" {
		pkgConfig.NameServer = c.NameServer
	}

	if c.TargetRoot != "" {
		pkgConfig.TargetRoot = c.TargetRoot
	}

	pkgConfig.RunConfig = c.RunConfig
	pkgConfig.CloudConfig = c.CloudConfig
	pkgConfig.Kernel = c.Kernel
	pkgConfig.Boot = c.Boot
	pkgConfig.Force = c.Force
	pkgConfig.NightlyBuild = c.NightlyBuild

	imageName := pkgConfig.RunConfig.Imagename
	images := path.Join(lepton.GetOpsHome(), "images")
	if imageName == "" {
		pkgConfig.RunConfig.Imagename = path.Join(images, filepath.Base(pkgConfig.Program))
		pkgConfig.CloudConfig.ImageName = fmt.Sprintf("%v-image", filepath.Base(pkgConfig.Program))
	} else {
		c.CloudConfig.ImageName = imageName
		imageName = path.Join(images, filepath.Base(imageName))
	}

	*c = *pkgConfig

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
