package lepton

import (
	"fmt"
	"os/exec"
	"strings"
)

// Datastores provides access to VSphere's Datastores
type Datastores struct{}

// CopyToBucket copies archive to bucket
func (s *Datastores) CopyToBucket(config *Config, archPath string) error {

	fmt.Printf("copying %v\n", archPath)

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
func (s *Datastores) DeleteFromBucket(config *Config, key string) error {
	return nil
}
