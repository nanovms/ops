//go:build relayered || !onlyprovider

package relayered

import (
	"github.com/nanovms/ops/types"
)

// Objects represent storage specific information for cloud object
// storage.
type Objects struct {
	token string
}

// CopyToBucket copies archive to bucket
func (s *Objects) CopyToBucket(config *types.Config, archPath string) error {
	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Objects) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
