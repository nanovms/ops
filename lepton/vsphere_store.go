package lepton

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nanovms/ops/config"
)

// Datastores provides access to VSphere's Datastores
type Datastores struct{}

// CopyToBucket converts the raw disk image to a monolithicFlat vmdk.
func (s *Datastores) CopyToBucket(config *config.Config, archPath string) error {

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
		fmt.Println(err)
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Datastores) DeleteFromBucket(config *config.Config, key string) error {
	return nil
}
