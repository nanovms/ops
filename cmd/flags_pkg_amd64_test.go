package cmd

import (
	"runtime"
	"testing"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestPkgFlagsPackagePath(t *testing.T) {
	packageName := "package-" + testutils.String(5)

	flagSet := newPkgFlagSet()
	flagSet.Set("local", "true")
	flagSet.Set("package", packageName)

	pkgFlags := NewPkgCommandFlags(flagSet)

	assert.Equal(t, pkgFlags.PackagePath(), api.GetOpsHome()+"/local_packages/"+runtime.GOARCH+"/"+packageName)
}
