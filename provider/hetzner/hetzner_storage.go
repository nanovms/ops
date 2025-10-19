package hetzner

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

const (
	objectStorageDomainEnv = "OBJECT_STORAGE_DOMAIN"
	objectStorageKeyEnv    = "OBJECT_STORAGE_KEY"
	objectStorageSecretEnv = "OBJECT_STORAGE_SECRET"
	defaultStorageDomain   = "your-objectstorage.com"
)

// ObjectStorage provides Hetzner object storage related operations.
type ObjectStorage struct{}

func (s *ObjectStorage) objectStorageDomain() string {
	if v := strings.TrimSpace(os.Getenv(objectStorageDomainEnv)); v != "" {
		return v
	}
	return defaultStorageDomain
}

func (s *ObjectStorage) getSignedURL(key, bucket, region string) string {
	client, err := s.newClient(region)
	if err != nil {
		log.Error(err)
		return ""
	}

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf(`attachment; filename="%s"`, key))

	url, err := client.PresignedGetObject(bucket, key, time.Second*5*60, reqParams)
	if err != nil {
		log.Error(err)
		return ""
	}

	return url.String()
}

func (s *ObjectStorage) getImageObjectStorageURL(config *types.Config, key string) string {
	bucket := config.CloudConfig.BucketName
	if bucket == "" {
		return ""
	}
	zone := strings.TrimSpace(config.CloudConfig.Zone)
	domain := s.objectStorageDomain()
	if zone == "" {
		return fmt.Sprintf("https://%s.%s/%s", bucket, domain, key)
	}
	return fmt.Sprintf("https://%s.%s.%s/%s", bucket, zone, domain, key)
}

// CopyToBucket copies archive to bucket.
func (s *ObjectStorage) CopyToBucket(config *types.Config, archPath string) error {
	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	client, err := s.getMinioClient(config)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	bucket := strings.TrimSpace(config.CloudConfig.BucketName)
	if bucket == "" {
		return fmt.Errorf("BucketName is required")
	}

	exists, err := client.BucketExists(bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", bucket)
	}

	key := filepath.Base(archPath)
	n, err := client.PutObject(bucket, key, file, stat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return err
	}

	log.Infof("uploaded %q (%d bytes) to bucket %q", key, n, bucket)

	policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::` + bucket + `/` + key + `"],"Sid": ""}]}`
	if err := client.SetBucketPolicy(bucket, policy); err != nil {
		return err
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket.
func (s *ObjectStorage) DeleteFromBucket(config *types.Config, key string) error {
	client, err := s.getMinioClient(config)
	if err != nil {
		return err
	}

	bucket := strings.TrimSpace(config.CloudConfig.BucketName)
	if bucket == "" {
		return fmt.Errorf("BucketName is required")
	}

	if err := client.RemoveObject(bucket, key); err != nil {
		return err
	}

	return nil
}

func (s *ObjectStorage) getMinioClient(config *types.Config) (*minio.Client, error) {
	zone := strings.TrimSpace(config.CloudConfig.Zone)
	return s.newClient(zone)
}

func (s *ObjectStorage) newClient(zone string) (*minio.Client, error) {
	accessKey := strings.TrimSpace(os.Getenv(objectStorageKeyEnv))
	if accessKey == "" {
		return nil, fmt.Errorf("set %s", objectStorageKeyEnv)
	}

	secretKey := strings.TrimSpace(os.Getenv(objectStorageSecretEnv))
	if secretKey == "" {
		return nil, fmt.Errorf("set %s", objectStorageSecretEnv)
	}

	domain := s.objectStorageDomain()
	endpoint := domain
	if zone != "" {
		endpoint = fmt.Sprintf("%s.%s", zone, domain)
	}

	return minio.New(endpoint, accessKey, secretKey, true)
}
