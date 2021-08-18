package aws

import (
	"fmt"
	"math"
	"os"

	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3 provides AWS storage related operations
type S3 struct{}

// CopyToBucket copies archive to bucket
func (s *S3) CopyToBucket(config *types.Config, archPath string) error {

	bucket := config.CloudConfig.BucketName
	zone := config.CloudConfig.Zone

	// verify we can even use the vm importer
	VerifyRole(zone, bucket)

	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(stripZone(zone))},
	)
	if err != nil {
		return err
	}

	fileStats, _ := file.Stat()
	log.Info("Uploading image with", fmt.Sprintf("%fMB", float64(fileStats.Size())/math.Pow(10, 6)))

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(config.CloudConfig.ImageName),
		Body:   file,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully uploaded %q to %q\n", config.CloudConfig.ImageName, bucket)

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (s *S3) DeleteFromBucket(config *types.Config, key string) error {
	bucket := config.CloudConfig.BucketName
	zone := config.CloudConfig.Zone

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(stripZone(zone))},
	)
	if err != nil {
		return err
	}
	svc := s3.New(sess)

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err = svc.DeleteObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				log.Error(aerr)
			}
		} else {
			log.Error(err)
		}
		return err
	}

	return nil
}
