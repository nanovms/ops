//go:build upcloud || !onlyprovider

package upcloud_test

import (
	"context"
	"testing"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/stretchr/testify/assert"
)

func TestCreateInstance(t *testing.T) {
	p, s := NewProvider(t)

	imageName := "test"
	instanceName := "test-1234"
	storageUUID := "1"
	serverID := "server-1"

	storages := upcloud.Storages{}
	storages.Storages = []upcloud.Storage{{UUID: storageUUID, Title: imageName}}

	s.EXPECT().
		GetStorages(context.Background(), &request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&storages, nil)

	s.EXPECT().
		CreateServer(context.Background(), &request.CreateServerRequest{
			Hostname: instanceName,
			Title:    instanceName,
			StorageDevices: request.CreateServerStorageDeviceSlice{
				{
					Title:   instanceName,
					Action:  "clone",
					Storage: storageUUID,
				},
			},
		}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID}}, nil)

	s.EXPECT().
		GetTags(context.Background()).
		Return(&upcloud.Tags{}, nil).
		Times(2)

	imageTag := upcloud.Tag{
		Name:        "image-test",
		Description: "Creted with image test",
	}

	createTagReq := &request.CreateTagRequest{Tag: imageTag}
	s.EXPECT().
		CreateTag(context.Background(), createTagReq).
		Return(&imageTag, nil)

	opsTag := upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}

	createTagReq = &request.CreateTagRequest{Tag: opsTag}
	s.EXPECT().
		CreateTag(context.Background(), createTagReq).
		Return(&opsTag, nil)

	s.EXPECT().
		TagServer(context.Background(), &request.TagServerRequest{UUID: serverID, Tags: []string{"OPS", "image-test"}}).
		Return(&upcloud.ServerDetails{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	ctx.Config().CloudConfig.ImageName = imageName
	ctx.Config().RunConfig.InstanceName = instanceName
	err := p.CreateInstance(ctx)

	assert.Nil(t, err)
}

func TestListInstances(t *testing.T) {
	p, s := NewProvider(t)

	tag := upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}
	s.EXPECT().
		GetTags(context.Background()).
		Return(&upcloud.Tags{Tags: []upcloud.Tag{tag}}, nil)

	s.EXPECT().
		GetServers(context.Background()).
		Return(&upcloud.Servers{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.ListInstances(ctx)

	assert.Nil(t, err)
}

func TestGetInstances(t *testing.T) {
	p, s := NewProvider(t)

	tag := upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}
	s.EXPECT().
		GetTags(context.Background()).
		Return(&upcloud.Tags{Tags: []upcloud.Tag{tag}}, nil)

	s.EXPECT().
		GetServers(context.Background()).
		Return(&upcloud.Servers{}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	instances, err := p.GetInstances(ctx)

	assert.Nil(t, err)

	assert.Equal(t, instances, []lepton.CloudInstance{})
}

func TestDeleteInstance(t *testing.T) {
	p, s := NewProvider(t)

	serverID := "server-1"
	serverName := "test"

	s.EXPECT().
		GetServers(context.Background()).
		Return(&upcloud.Servers{Servers: []upcloud.Server{{UUID: serverID, Title: serverName}}}, nil)

	s.EXPECT().
		GetServerDetails(context.Background(), &request.GetServerDetailsRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		StopServer(context.Background(), &request.StopServerRequest{UUID: serverID}).
		Return(nil, nil)

	s.EXPECT().
		WaitForServerState(context.Background(), &request.WaitForServerStateRequest{UUID: serverID, DesiredState: "stopped", Timeout: 1 * time.Minute}).
		Return(nil, nil)

	s.EXPECT().
		DeleteServer(context.Background(), &request.DeleteServerRequest{UUID: serverID}).
		Return(nil)

	ctx := lepton.NewContext(lepton.NewConfig())
	err := p.DeleteInstance(ctx, serverName)

	assert.Nil(t, err)
}
