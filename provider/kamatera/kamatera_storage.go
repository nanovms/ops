package kamatera

import (
	"os"

	"github.com/nanovms/ops/types"
)

// ObjectStorage provides Kamatera object storage related operations.
type ObjectStorage struct{}

// CopyToBucket copies archive to bucket.
func (s *ObjectStorage) CopyToBucket(config *types.Config, archPath string) error {
	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

// DeleteFromBucket deletes key from config's bucket.
func (s *ObjectStorage) DeleteFromBucket(config *types.Config, key string) error {
	return nil
}
