package lepton

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go"
	"github.com/nanovms/ops/config"
)

// Objects provides Vultr Object Storage related operations
type Objects struct{}

func (s *Objects) getSignedURL(key string, bucket string, region string) string {
	accessKey := os.Getenv("VULTR_ACCESS")
	secKey := os.Getenv("VULTR_SECRET")

	if accessKey == "" || secKey == "" {
		fmt.Println("danger will robinson - can not find VULTR_ACCESS || VULTR_SECRET env vars")
		os.Exit(1)
	}

	endpoint := region + ".vultrobjects.com"

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

// CopyToBucket copies archive to bucket
func (s *Objects) CopyToBucket(config *config.Config, archPath string) error {

	bucket := config.CloudConfig.BucketName
	zone := config.CloudConfig.Zone

	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	accessKey := os.Getenv("VULTR_ACCESS")
	secKey := os.Getenv("VULTR_SECRET")

	endpoint := zone + ".vultrobjects.com"

	ssl := true

	client, err := minio.New(endpoint, accessKey, secKey, ssl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	stat, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	n, err := client.PutObject(bucket, config.CloudConfig.ImageName+".img", file, stat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Uploaded", "my-objectname", " of size: ", n, "Successfully.")

	fmt.Printf("Successfully uploaded %q to %q\n", config.CloudConfig.ImageName, bucket)

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *Objects) DeleteFromBucket(config *config.Config, key string) error {
	return nil
}
