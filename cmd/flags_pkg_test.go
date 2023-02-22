package cmd_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"

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
	packageName := "package-" + testutils.String(5)

	flagSet := newPkgFlagSet()
	flagSet.Set("local", "true")
	flagSet.Set("package", packageName)

	pkgFlags := cmd.NewPkgCommandFlags(flagSet)

	assert.Equal(t, pkgFlags.PackagePath(), api.GetOpsHome()+"/local_packages/"+packageName)
}

func TestPkgFlagsMergeToConfig(t *testing.T) {
	packageName := "package-" + testutils.String(5)

	pkgConfig := &types.Config{
		Program:      "ops",
		Args:         []string{"a", "b"},
		Dirs:         []string{"da", "db"},
		Files:        []string{"fa", "fb"},
		MapDirs:      map[string]string{"a": "/patha", "b": "/pathb"},
		Env:          map[string]string{"a": "vala", "b": "valb"},
		BaseVolumeSz: "100m",
		NameServers:  []string{"manifest.ops.net"},
		TargetRoot:   "manifest-unix",
		Kernel:       "/manifest/path/to/kernel",
		Boot:         "/manifest/path/to/boot",
		Force:        false,
		NightlyBuild: false,
		CloudConfig: types.ProviderConfig{
			ProjectID:  "manifest-projectid",
			BucketName: "manifest-thebucketname",
		},
		RunConfig: types.RunConfig{
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
		fmt.Printf("Failed to create dir %s, error is: %s", pkgFlags.PackagePath(), err)
		os.Exit(1)
	}
	err = os.Chmod(pkgFlags.PackagePath(), 0777)
	if err != nil {
		fmt.Printf("Failed to chmod dir %s, error is: %s", pkgFlags.PackagePath(), err)
		os.Exit(1)
	}
	writeConfigToFile(pkgConfig, manifestPath)
	defer os.RemoveAll(pkgFlags.PackagePath())

	c := &types.Config{
		Args:         []string{"c", "d"},
		Dirs:         []string{"dc", "dd"},
		Files:        []string{"fc", "fd"},
		MapDirs:      map[string]string{"c": "/pathc", "d": "/pathd"},
		Env:          map[string]string{"c": "valc", "d": "vald"},
		BaseVolumeSz: "200m",
		NameServers:  []string{"ops.net"},
		TargetRoot:   "unix",
		Kernel:       "/path/to/kernel",
		Boot:         "/path/to/boot",
		Force:        true,
		NightlyBuild: true,
		CloudConfig: types.ProviderConfig{
			ProjectID:  "projectid",
			BucketName: "thebucketname",
		},
		RunConfig: types.RunConfig{
			Memory: "2G",
		},
	}

	err = pkgFlags.MergeToConfig(c)

	assert.Nil(t, err)

	expectedConfig := &types.Config{
		Program:      "ops",
		Args:         []string{"a", "b", "c", "d"},
		Dirs:         []string{"da", "db", "dc", "dd"},
		Files:        []string{"fa", "fb", "fc", "fd"},
		MapDirs:      map[string]string{"a": "/patha", "b": "/pathb", "c": "/pathc", "d": "/pathd"},
		Env:          map[string]string{"a": "vala", "b": "valb", "c": "valc", "d": "vald"},
		BaseVolumeSz: "200m",
		NameServers:  []string{"ops.net"},
		TargetRoot:   "unix",
		Kernel:       "/path/to/kernel",
		Boot:         "/path/to/boot",
		Force:        true,
		NightlyBuild: true,
		CloudConfig: types.ProviderConfig{
			ProjectID:  "projectid",
			BucketName: "thebucketname",
			ImageName:  "ops-image",
		},
		RunConfig: types.RunConfig{
			Memory:    "2G",
			ImageName: lepton.GetOpsHome() + "/images/ops",
		},
	}

	assert.Equal(t, expectedConfig, c)
}

func newPkgFlagSet() (flagSet *pflag.FlagSet) {
	flagSet = pflag.NewFlagSet("test", 0)

	cmd.PersistPkgCommandFlags(flagSet)
	return
}
