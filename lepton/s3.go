package lepton

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3 provides AWS storage related operations
type S3 struct{}

// CopyToBucket copies archive to bucket
func (s *S3) CopyToBucket(config *Config, archPath string) error {

	bucket := config.CloudConfig.BucketName
	zone := "us-west-1"

	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(zone)},
	)
	if err != nil {
		return err
	}

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
func (s *S3) DeleteFromBucket(config *Config, key string) error {
	bucket := config.CloudConfig.BucketName
	zone := "us-west-1"

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(zone)},
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
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}
