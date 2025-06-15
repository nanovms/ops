package oci

import (
	"os"
	"path"
	"testing"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
	"gotest.tools/assert"
)

func TestBuildImage(t *testing.T) {
	p, _, _, _, _, _, _ := NewProvider(t)

	ctx := lepton.NewContext(lepton.NewConfig())

	imageName := "oci-build-image-test"

	ctx.Config().CloudConfig.ImageName = imageName
	ctx.Config().RunConfig.ImageName = imageName
	ctx.Config().Program = testutils.BuildBasicProgram()
	defer os.Remove(ctx.Config().Program)

	qcow2ImagePath, err := p.BuildImage(ctx)

	assert.NilError(t, err)

	assert.Equal(t, qcow2ImagePath, path.Join(lepton.GetOpsHome(), "qcow2-images", imageName+".qcow2"))

	os.Remove(qcow2ImagePath)
}
