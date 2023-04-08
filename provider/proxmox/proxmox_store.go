//go:build proxmox || !onlyprovider

package proxmox

import (
	"github.com/nanovms/ops/types"
)

// Objects provides ProxMoxr Object Storage related operations
type Objects struct{}

// CopyToBucket copies archive to bucket
func (s *Objects) CopyToBucket(config *types.Config, archPath string) error {
	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Objects) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
