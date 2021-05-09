package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	storage "cloud.google.com/go/storage"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// Storage provides GCP storage related operations
type Storage struct{}

// CopyToBucket copies archive to bucket
func (s *Storage) CopyToBucket(config *types.Config, archPath string) error {

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error(err.Error())
		log.Fatal("Have you set GOOGLE_APPLICATION_CREDENTIALS?")
	}

	defer client.Close()

	bucket := client.Bucket(config.CloudConfig.BucketName)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		// Creates the new bucket.
		log.Info("creating bucket:", config.CloudConfig.BucketName)
		if err := bucket.Create(ctx, config.CloudConfig.ProjectID, nil); err != nil {
			return fmt.Errorf("failed to create bucket: %+v", err)
		}
	} else {
		log.Info("bucket found:", config.CloudConfig.BucketName)
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
