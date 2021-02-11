package cmd_test

import (
	"os"
	"testing"

	"github.com/nanovms/ops/cmd"
	"github.com/nanovms/ops/lepton"
	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestBuildPkgFlags(t *testing.T) {
	flagset := newPkgFlagSet()

	flagset.Set("local", "true")
	flagset.Set("package", "mysql")

	pkgFlags := cmd.NewPkgCommandFlags(flagset)

	assert.Equal(t, pkgFlags, &cmd.PkgCommandFlags{
		LocalPackage: true,
		Package:      "mysql",
	})
}

func TestPkgFlagsPackagePath(t *testing.T) {
	packageName := "package-" + String(5)

	flagSet := newPkgFlagSet()
	flagSet.Set("local", "true")
	flagSet.Set("package", packageName)

	pkgFlags := cmd.NewPkgCommandFlags(flagSet)

	assert.Equal(t, pkgFlags.PackagePath(), api.GetOpsHome()+"/local_packages/"+packageName)
}

func TestPkgFlagsMergeToConfig(t *testing.T) {
	packageName := "package-" + String(5)

	pkgConfig := &lepton.Config{
		Program:      "ops",
		Args:         []string{"a", "b"},
		Dirs:         []string{"da", "db"},
		Files:        []string{"fa", "fb"},
		MapDirs:      map[string]string{"a": "/patha", "b": "/pathb"},
		Env:          map[string]string{"a": "vala", "b": "valb"},
		BaseVolumeSz: "100m",
		NameServer:   "manifest.ops.net",
		TargetRoot:   "manifest-unix",
		Kernel:       "/manifest/path/to/kernel",
		Boot:         "/manifest/path/to/boot",
		Force:        false,
		NightlyBuild: false,
		CloudConfig: lepton.ProviderConfig{
			ProjectID:  "manifest-projectid",
			BucketName: "manifest-thebucketname",
		},
		RunConfig: lepton.RunConfig{
			Memory: "manifest-2G",
		},
	}

	flagSet := newPkgFlagSet()
	flagSet.Set("local", "true")
	flagSet.Set("package", packageName)

	pkgFlags := cmd.NewPkgCommandFlags(flagSet)

	manifestPath := pkgFlags.PackagePath() + "/package.manifest"

	err := os.Mkdir(pkgFlags.PackagePath(), 0666)
	if err != nil {
		panic(err)
	}
	err = os.Chmod(pkgFlags.PackagePath(), 0777)
	if err != nil {
		panic(err)
	}
	writeConfigToFile(pkgConfig, manifestPath)
	defer os.RemoveAll(pkgFlags.PackagePath())

	config := &lepton.Config{
		Args:         []string{"c", "d"},
		Dirs:         []string{"dc", "dd"},
		Files:        []string{"fc", "fd"},
		MapDirs:      map[string]string{"c": "/pathc", "d": "/pathd"},
		Env:          map[string]string{"c": "valc", "d": "vald"},
		BaseVolumeSz: "200m",
		NameServer:   "ops.net",
		TargetRoot:   "unix",
		Kernel:       "/path/to/kernel",
		Boot:         "/path/to/boot",
		Force:        true,
		NightlyBuild: true,
		CloudConfig: lepton.ProviderConfig{
			ProjectID:  "projectid",
			BucketName: "thebucketname",
		},
		RunConfig: lepton.RunConfig{
			Memory: "2G",
		},
	}

	err = pkgFlags.MergeToConfig(config)

	assert.Nil(t, err)

	expectedConfig := &lepton.Config{
		Program: "ops",
		Args:    []string{"a", "b", "c", "d"},
		Dirs:    []string{"da", "db", "dc", "dd"},
		Files:   []string{"fa", "fb", "fc", "fd"},
		MapDirs: map[string]string{"a": "/patha", "b": "/pathb", "c": "/pathc", "d": "/pathd"},
		Env:     map[string]string{"a": "vala", "b": "valb", "c": "valc", "d": "vald"}, BaseVolumeSz: "200m",
		NameServer:   "ops.net",
		TargetRoot:   "unix",
		Kernel:       "/path/to/kernel",
		Boot:         "/path/to/boot",
		Force:        true,
		NightlyBuild: true,
		CloudConfig: lepton.ProviderConfig{
			ProjectID:  "projectid",
			BucketName: "thebucketname",
			ImageName:  "ops-image",
		},
		RunConfig: lepton.RunConfig{
			Memory:    "2G",
			Imagename: lepton.GetOpsHome() + "/images/ops",
		},
	}

	assert.Equal(t, expectedConfig, config)
}

func newPkgFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistPkgCommandFlags(flagSet)
	return
}
