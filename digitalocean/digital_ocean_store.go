package digitalocean

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go"
	"github.com/nanovms/ops/config"
)

// Spaces provides Digital Ocean storage related operations
type Spaces struct{}

func (s *Spaces) getSignedURL(key string, bucket string, region string) string {
	accessKey := os.Getenv("SPACES_KEY")
	secKey := os.Getenv("SPACES_SECRET")

	endpoint := region + ".digitaloceanspaces.com"

	ssl := true

	client, err := minio.New(endpoint, accessKey, secKey, ssl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "attachment; filename=\""+key+"\"")
	presignedURL, err := client.PresignedGetObject(bucket, key, time.Second*5*60, reqParams)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	return presignedURL.String()
}

func (s *Spaces) getImageSpacesURL(config *config.Config, imageName string) string {
	return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", config.CloudConfig.BucketName, config.CloudConfig.Zone, imageName)
}

// CopyToBucket copies archive to bucket
func (s *Spaces) CopyToBucket(config *config.Config, archPath string) error {

	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	client, err := s.getMinioClient(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	stat, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	bucket := config.CloudConfig.BucketName
	key := filepath.Base(archPath)

	n, err := client.PutObject(bucket, key, file, stat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Uploaded", "my-objectname", " of size: ", n, "Successfully.")

	fmt.Printf("Successfully uploaded %q to %q\n", config.CloudConfig.ImageName, bucket)

	policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::ops/` + key + `"],"Sid": ""}]}`

	err = client.SetBucketPolicy(bucket, policy)
	if err != nil {
		return nil
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Spaces) DeleteFromBucket(config *config.Config, key string) error {
	bucket := config.CloudConfig.BucketName

	client, err := s.getMinioClient(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = client.RemoveObject(bucket, key)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (s *Spaces) getMinioClient(config *config.Config) (*minio.Client, error) {
	zone := config.CloudConfig.Zone

	accessKey := os.Getenv("SPACES_KEY")
	secKey := os.Getenv("SPACES_SECRET")

	endpoint := zone + ".digitaloceanspaces.com"

	ssl := true

	return minio.New(endpoint, accessKey, secKey, ssl)
}
