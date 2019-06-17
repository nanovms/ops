package lepton

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	storage "cloud.google.com/go/storage"
)

// GCPStorage provides GCP storage related operations
type GCPStorage struct{}

// CopyToBucket copies archieve to bucket
func (s *GCPStorage) CopyToBucket(config *Config, archPath string) error {

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	defer client.Close()

	if err != nil {
		return fmt.Errorf("failed to create client: %+v", err)
	}

	bucket := client.Bucket(config.CloudConfig.BucketName)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		// Creates the new bucket.
		fmt.Println("creating bucket:", config.CloudConfig.BucketName)
		if err := bucket.Create(ctx, config.CloudConfig.ProjectID, nil); err != nil {
			return fmt.Errorf("failed to create bucket: %+v", err)
		}
	} else {
		fmt.Println("bucket found:", config.CloudConfig.BucketName)
	}

	wr := bucket.Object(filepath.Base(archPath)).NewWriter(ctx)
	f, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(wr, f); err != nil {
		return err
	}
	if err = wr.Close(); err != nil {
		return err
	}
	return nil
}
