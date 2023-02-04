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

func TestGetAllVolumes(t *testing.T) {
	p, s := NewProvider(t)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Type: "disk", Access: "private"}).
		Return(&upcloud.Storages{}, nil)

	volumes, err := p.GetAllVolumes(testutils.NewMockContext())

	assert.Nil(t, err)

	assert.Equal(t, volumes, &[]lepton.NanosVolume{})
}

func TestAttachVolume(t *testing.T) {
	p, s := NewProvider(t)

	serverID := "server-1"
	serverName := "test"
	volumeID := "volume-1"
	volumeName := "files"

	s.EXPECT().
		GetServers().
		Return(&upcloud.Servers{Servers: []upcloud.Server{{UUID: serverID, Title: serverName}}}, nil)

	s.EXPECT().
		GetServerDetails(&request.GetServerDetailsRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		StopServer(&request.StopServerRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		WaitForServerState(&request.WaitForServerStateRequest{UUID: serverID, DesiredState: "stopped", Timeout: 1 * time.Minute}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Type: "disk", Access: "private"}).
		Return(&upcloud.Storages{Storages: []upcloud.Storage{{UUID: volumeID, Title: volumeName}}}, nil)

	s.EXPECT().
		GetStorageDetails(&request.GetStorageDetailsRequest{UUID: volumeID}).
		Return(&upcloud.StorageDetails{Storage: upcloud.Storage{UUID: volumeID, Title: volumeName}}, nil)

	s.EXPECT().
		AttachStorage(&request.AttachStorageRequest{ServerUUID: serverID, StorageUUID: volumeID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID}}, nil)

	s.EXPECT().
		StartServer(&request.StartServerRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	err := p.AttachVolume(testutils.NewMockContext(), serverName, volumeName, 1)

	assert.Nil(t, err)
}

func TestDetachVolume(t *testing.T) {
	p, s := NewProvider(t)

	serverID := "server-1"
	serverName := "test"
	volumeID := "volume-1"
	volumeName := "files"

	s.EXPECT().
		GetServers().
		Return(&upcloud.Servers{Servers: []upcloud.Server{{UUID: serverID, Title: serverName}}}, nil)

	s.EXPECT().
		GetServerDetails(&request.GetServerDetailsRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{StorageDevices: []upcloud.ServerStorageDevice{{Title: volumeName, Address: "s0"}}, Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		StopServer(&request.StopServerRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		WaitForServerState(&request.WaitForServerStateRequest{UUID: serverID, DesiredState: "stopped", Timeout: 1 * time.Minute}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	s.EXPECT().
		GetStorages(&request.GetStoragesRequest{Type: "disk", Access: "private"}).
		Return(&upcloud.Storages{Storages: []upcloud.Storage{{UUID: volumeID, Title: volumeName}}}, nil)

	s.EXPECT().
		GetStorageDetails(&request.GetStorageDetailsRequest{UUID: volumeID}).
		Return(&upcloud.StorageDetails{Storage: upcloud.Storage{UUID: volumeID, Title: volumeName}}, nil)

	s.EXPECT().
		DetachStorage(&request.DetachStorageRequest{ServerUUID: serverID, Address: "s0"}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID}}, nil)

	s.EXPECT().
		StartServer(&request.StartServerRequest{UUID: serverID}).
		Return(&upcloud.ServerDetails{Server: upcloud.Server{UUID: serverID, Title: serverName}}, nil)

	err := p.DetachVolume(testutils.NewMockContext(), serverName, volumeName)

	assert.Nil(t, err)
}
