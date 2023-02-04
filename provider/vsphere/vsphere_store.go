package vsphere

import (
	"os/exec"
	"strings"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// Datastores provides access to VSphere's Datastores
type Datastores struct{}

// CopyToBucket converts the raw disk image to a monolithicFlat vmdk.
func (s *Datastores) CopyToBucket(config *types.Config, archPath string) error {

	vmdkPath := "/tmp/" + config.CloudConfig.ImageName + ".vmdk"

	vmdkPath = strings.ReplaceAll(vmdkPath, "-image", "")

	args := []string{
		"convert", "-f", "raw",
		"-O", "vmdk", "-o", "subformat=monolithicFlat",
		archPath, vmdkPath,
	}

	cmd := exec.Command("qemu-img", args...)
	err := cmd.Run()
	if err != nil {
		log.Error(err)
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Datastores) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
