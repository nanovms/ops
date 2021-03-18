package upcloud_test

import (
	"testing"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
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
		GetStorages(&request.GetStoragesRequest{Access: "private", Type: "template"}).
		Return(&storages, nil)

	s.EXPECT().
		CreateServer(&request.CreateServerRequest{
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
		GetTags().
		Return(&upcloud.Tags{}, nil).
		Times(2)

	imageTag := upcloud.Tag{
		Name:        "image-test",
		Description: "Creted with image test",
	}

	createTagReq := &request.CreateTagRequest{Tag: imageTag}
	s.EXPECT().
		CreateTag(createTagReq).
		Return(&imageTag, nil)

	opsTag := upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}

	createTagReq = &request.CreateTagRequest{Tag: opsTag}
	s.EXPECT().
		CreateTag(createTagReq).
		Return(&opsTag, nil)

	s.EXPECT().
		TagServer(&request.TagServerRequest{UUID: serverID, Tags: []string{"OPS", "image-test"}}).
		Return(&upcloud.ServerDetails{}, nil)

	ctx := testutils.NewMockContext()
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
		GetTags().
		Return(&upcloud.Tags{Tags: []upcloud.Tag{tag}}, nil)

	s.EXPECT().
		GetServers().
		Return(&upcloud.Servers{}, nil)

	err := p.ListInstances(testutils.NewMockContext())

	assert.Nil(t, err)
}

func TestGetInstances(t *testing.T) {
	p, s := NewProvider(t)

	tag := upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}
	s.EXPECT().
		GetTags().
		Return(&upcloud.Tags{Tags: []upcloud.Tag{tag}}, nil)

	s.EXPECT().
		GetServers().
		Return(&upcloud.Servers{}, nil)

	instances, err := p.GetInstances(testutils.NewMockContext())

	assert.Nil(t, err)

	assert.Equal(t, instances, []lepton.CloudInstance{})
}

func TestDeleteInstance(t *testing.T) {
	p, s := NewProvider(t)

	serverID := "server-1"
	serverName := "test"

	s.EXPECT().
		GetServers().
		Return(&upcloud.Servers{Servers: []upcloud.Server{{UUID: serverID, Title: serverName}}}, nil)

	s.EXPECT().
		GetServerDetails(&request.GetServerDetailsRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		StopServer(&request.StopServerRequest{UUID: serverID}).
		Return(nil, nil)

	s.EXPECT().
		WaitForServerState(&request.WaitForServerStateRequest{UUID: serverID, DesiredState: "stopped", Timeout: 1 * time.Minute}).
		Return(nil, nil)

	s.EXPECT().
		DeleteServer(&request.DeleteServerRequest{UUID: serverID}).
		Return(nil)

	err := p.DeleteInstance(testutils.NewMockContext(), serverName)

	assert.Nil(t, err)
}
