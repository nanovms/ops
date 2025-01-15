//go:build aws || !onlyprovider

package aws

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// S3 provides AWS storage related operations
type S3 struct{}

// CopyToBucket copies archive to bucket
func (s *S3) CopyToBucket(config *types.Config, archPath string) error {

	bucket := config.CloudConfig.BucketName
	execCtx := context.Background()
	awsSdkConfig, err := GetAwsSdkConfig(execCtx, &config.CloudConfig.Zone)

	// this verification/role creator can be skipped for users that
	// already have it setup but don't have rights to verify
	if !config.CloudConfig.SkipImportVerify {
		if err != nil {
			return err
		}
		iamClient := iam.NewFromConfig(*awsSdkConfig)
		// verify we can even use the vm importer
		VerifyRole(execCtx, iamClient, config.CloudConfig.Zone, bucket)
	}

	file, err := os.Open(archPath)
	if err != nil {
		return err
	}
	defer file.Close()

	s3Client := s3.NewFromConfig(*awsSdkConfig)

	fileStats, _ := file.Stat()
	log.Info("Uploading image with", fmt.Sprintf("%fMB", float64(fileStats.Size())/math.Pow(10, 6)))

	uploader := manager.NewUploader(s3Client)
	_, err = uploader.Upload(execCtx, &s3.PutObjectInput{
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
func (s *S3) DeleteFromBucket(execCtx context.Context, config *types.Config, key string) error {
	bucket := config.CloudConfig.BucketName

	awsSdkConfig, err := GetAwsSdkConfig(execCtx, &config.CloudConfig.Zone)
	if err != nil {
		return err
	}
	s3Client := s3.NewFromConfig(*awsSdkConfig)

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err = s3Client.DeleteObject(execCtx, input)
	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			switch aerr.ErrorCode() {
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
