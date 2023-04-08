//go:build upcloud || !onlyprovider

package upcloud_test

import (
	"os"
	"testing"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	p, s := NewProvider(t)

	file, _ := os.CreateTemp("/tmp", "test-path")
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

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.CreateImage(ctx, file.Name())

	assert.Nil(t, err)
}

func TestListImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&upcloud.Storages{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.ListImages(ctx)

	assert.Nil(t, err)
}

func TestGetImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&upcloud.Storages{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	images, err := p.GetImages(ctx)

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

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.DeleteImage(ctx, storageTitle)

	assert.Nil(t, err)
}
