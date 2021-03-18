package oci_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nanovms/ops/mock_oci"
	"github.com/nanovms/ops/oci"
	"github.com/spf13/afero"
)

var (
	ociOpsTags = map[string]string{"CreatedBy": "OPS"}
)

func createTags(tags map[string]string) map[string]string {
	for k, v := range ociOpsTags {
		tags[k] = v
	}

	return tags
}

func NewProvider(t *testing.T) (*oci.Provider, *mock_oci.MockComputeService, *mock_oci.MockStorageService, *mock_oci.MockWorkRequestService, *mock_oci.MockNetworkService, *mock_oci.MockBlockstorageService, afero.Fs) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	computeClient := mock_oci.NewMockComputeService(ctrl)
	storageClient := mock_oci.NewMockStorageService(ctrl)
	workRequestClient := mock_oci.NewMockWorkRequestService(ctrl)
	networkClient := mock_oci.NewMockNetworkService(ctrl)
	blockstorageClient := mock_oci.NewMockBlockstorageService(ctrl)
	fileReader := afero.NewMemMapFs()

	return oci.NewProviderWithClients(computeClient, storageClient, workRequestClient, networkClient, blockstorageClient, fileReader), computeClient, storageClient, workRequestClient, networkClient, blockstorageClient, fileReader
}
