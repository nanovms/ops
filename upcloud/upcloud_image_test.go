package upcloud_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	p, s := NewProvider(t)

	file, _ := ioutil.TempFile("/tmp", "test-path")
	defer os.Remove(file.Name())

	storageUUID := "1"

	s.EXPECT().
		CreateStorage(&request.CreateStorageRequest{Size: 10}).
		Return(&upcloud.StorageDetails{Storage: upcloud.Storage{UUID: storageUUID}}, nil)

	s.EXPECT().
		CreateStorageImport(&request.CreateStorageImportRequest{StorageUUID: storageUUID, Source: "direct_upload", SourceLocation: file.Name()}).
		Return(&upcloud.StorageImportDetails{}, nil)

	s.EXPECT().
		WaitForStorageState(&request.WaitForStorageStateRequest{UUID: storageUUID, DesiredState: "online", Timeout: 10 * time.Minute}).
		Return(nil, nil).
		Times(2)

	s.EXPECT().
		TemplatizeStorage(&request.TemplatizeStorageRequest{UUID: storageUUID}).
		Return(nil, nil)

	s.EXPECT().
		DeleteStorage(&request.DeleteStorageRequest{UUID: storageUUID}).
		Return(nil)

	err := p.CreateImage(testutils.NewMockContext(), file.Name())

	assert.Nil(t, err)
}

func TestListImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&upcloud.Storages{}, nil)

	err := p.ListImages(testutils.NewMockContext())

	assert.Nil(t, err)
}

func TestGetImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&upcloud.Storages{}, nil)

	images, err := p.GetImages(testutils.NewMockContext())

	assert.Nil(t, err)

	assert.Equal(t, images, []lepton.CloudImage{})
}

func TestDeleteImage(t *testing.T) {
	p, s := NewProvider(t)

	storageUUID := "1"
	storageTitle := "test"

	storages := upcloud.Storages{}

	storages.Storages = []upcloud.Storage{{UUID: storageUUID, Title: storageTitle}}

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&storages, nil)

	s.EXPECT().
		DeleteStorage(&request.DeleteStorageRequest{UUID: storageUUID}).
		Return(nil)

	err := p.DeleteImage(testutils.NewMockContext(), storageTitle)

	assert.Nil(t, err)
}
