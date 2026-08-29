//go:build oci || !onlyprovider

package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/oracle/oci-go-sdk/v65/objectstorage/transfer"
	"github.com/spf13/afero"
)

// ImageUploader puts a local image file into an Object Storage bucket. It is a seam because the
// SDK's upload manager wants the concrete ObjectStorageClient, which the StorageService interface
// deliberately hides.
type ImageUploader interface {
	UploadImage(ctx context.Context, namespace, bucket, object, filePath string, progress func(part, total int)) error
}

// uploadPartSize is the size of each part of a multipart upload. Object Storage allows up to 10000
// parts, so 32 MiB covers any image ops can build while keeping a failed part cheap to retry.
const uploadPartSize = 16 * 1024 * 1024

// uploadConcurrency is how many parts travel at once. Three saturates an ordinary uplink without
// starving the retries of the parts already in flight.
const uploadConcurrency = 3

// UploadPartTimeout is how long a single part may take. The SDK's HTTP client applies its timeout
// to the whole request, sending the body included, and defaults it to 60 seconds, which no part of
// a useful size meets on an ordinary uplink, all the more so with several parts sharing it:
//
//	Put ".../u/<image>?uploadId=...&uploadPartNum=3": context deadline exceeded
//	(Client.Timeout exceeded while awaiting headers)
//
// It stays a bound, not an absence of one: a part that stops making progress still fails instead of
// hanging forever.
const UploadPartTimeout = 30 * time.Minute

// multipartUploader uploads through the SDK's upload manager, which splits the file into parts,
// sends them in parallel and retries a single failed part instead of the whole transfer. This is
// what the OCI CLI does, and the reason a single PutObject is not enough: it sends the entire image
// in one request and the SDK's HTTP client gives up on its own timeout while still awaiting the
// response headers, which is what a multi-gigabyte image on an ordinary uplink always does:
//
//	Put ".../o/<image>": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
type multipartUploader struct {
	client  *objectstorage.ObjectStorageClient
	manager *transfer.UploadManager
}

func (u *multipartUploader) UploadImage(ctx context.Context, namespace, bucket, object, filePath string, progress func(part, total int)) error {
	_, err := u.manager.UploadFile(ctx, transfer.UploadFileRequest{
		UploadRequest: transfer.UploadRequest{
			NamespaceName:                       &namespace,
			BucketName:                          &bucket,
			ObjectName:                          &object,
			ObjectStorageClient:                 u.client,
			EnableMultipartChecksumVerification: common.Bool(true),
			PartSize:                            common.Int64(uploadPartSize),
			NumberOfGoroutines:                  common.Int(uploadConcurrency),
			CallBack: func(part transfer.MultiPartUploadPart) {
				if part.Err == nil && progress != nil {
					progress(part.PartNum, part.TotalParts)
				}
			},
		},
		FilePath: filePath,
	})

	return err
}

// singlePutUploader sends the whole object in one PutObject through the StorageService seam. It is
// the uploader a Provider built with injected clients gets, so tests keep exercising that seam.
type singlePutUploader struct {
	storage    StorageService
	fileSystem afero.Fs
}

func (u *singlePutUploader) UploadImage(ctx context.Context, namespace, bucket, object, filePath string, progress func(part, total int)) error {
	image, err := u.fileSystem.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed reading file %s: %w", filePath, err)
	}
	defer image.Close()

	stats, err := image.Stat()
	if err != nil {
		return fmt.Errorf("failed getting file stats of %s: %w", filePath, err)
	}

	size := stats.Size()

	_, err = u.storage.PutObject(ctx, objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    &bucket,
		ContentLength: &size,
		ObjectName:    &object,
		PutObjectBody: image,
	})

	return err
}
