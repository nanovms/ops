//go:build upcloud || !onlyprovider

package upcloud_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nanovms/ops/lepton"

	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/request"
	"github.com/stretchr/testify/assert"
)

func TestCreateImage(t *testing.T) {
	p, s := NewProvider(t)

	file, _ := os.CreateTemp("/tmp", "test-path")
	defer os.Remove(file.Name())

	storageUUID := "1"

	s.EXPECT().
		CreateStorage(context.Background(), &request.CreateStorageRequest{Size: 1}).
		Return(&upcloud.StorageDetails{Storage: upcloud.Storage{UUID: storageUUID}}, nil)

	s.EXPECT().
		CreateStorageImport(context.Background(), &request.CreateStorageImportRequest{StorageUUID: storageUUID, Source: "direct_upload", SourceLocation: file.Name()}).
		Return(&upcloud.StorageImportDetails{}, nil)

	s.EXPECT().
		WaitForStorageState(context.Background(), &request.WaitForStorageStateRequest{UUID: storageUUID, DesiredState: "online", Timeout: 10 * time.Minute}).
		Return(nil, nil).
		Times(2)

	s.EXPECT().
		TemplatizeStorage(context.Background(), &request.TemplatizeStorageRequest{UUID: storageUUID}).
		Return(nil, nil)

	s.EXPECT().
		DeleteStorage(context.Background(), &request.DeleteStorageRequest{UUID: storageUUID}).
		Return(nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.CreateImage(ctx, file.Name())

	assert.Nil(t, err)
}

func TestListImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(context.Background(), &request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&upcloud.Storages{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.ListImages(ctx)

	assert.Nil(t, err)
}

func TestGetImages(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(context.Background(), &request.GetStoragesRequest{Access: "private", Type: "template"}).
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
		GetStorages(context.Background(), &request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&storages, nil)

	s.EXPECT().
		DeleteStorage(context.Background(), &request.DeleteStorageRequest{UUID: storageUUID}).
		Return(nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.DeleteImage(ctx, storageTitle)

	assert.Nil(t, err)
}
