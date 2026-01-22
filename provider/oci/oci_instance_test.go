//go:build oci || !onlyprovider

package oci_test

import (
	"context"
	"testing"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"gotest.tools/assert"
)

func TestGetInstances(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := lepton.NewContext(lepton.NewConfig())

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	instances, err := p.GetInstances(ctx)

	assert.NilError(t, err)

	expected := []lepton.CloudInstance{
		{ID: "1", Status: "RUNNING", Name: "instance-1", Created: "a long while ago"},
		{ID: "2", Status: "STOPPED", Name: "instance-2", Created: "a long while ago"},
	}

	assert.DeepEqual(t, expected, instances)
}

func TestGetInstanceByID(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := lepton.NewContext(lepton.NewConfig())

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	instance, err := p.GetInstanceByName(ctx, "instance-2")

	assert.NilError(t, err)

	expected := &lepton.CloudInstance{ID: "2", Status: "STOPPED", Name: "instance-2", Created: "a long while ago"}

	assert.DeepEqual(t, expected, instance)
}

func TestDeleteInstance(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := lepton.NewContext(lepton.NewConfig())

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	c.EXPECT().
		TerminateInstance(context.TODO(), core.TerminateInstanceRequest{InstanceId: types.StringPtr("2")}).
		Return(core.TerminateInstanceResponse{}, nil)

	err := p.DeleteInstance(ctx, "instance-2")

	assert.NilError(t, err)
}

func TestStartInstance(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := lepton.NewContext(lepton.NewConfig())

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	c.EXPECT().
		InstanceAction(context.TODO(), core.InstanceActionRequest{Action: core.InstanceActionActionStart, InstanceId: types.StringPtr("2")}).
		Return(core.InstanceActionResponse{}, nil)

	err := p.StartInstance(ctx, "instance-2")

	assert.NilError(t, err)
}

func TestStopInstance(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := lepton.NewContext(lepton.NewConfig())

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	c.EXPECT().
		InstanceAction(context.TODO(), core.InstanceActionRequest{Action: core.InstanceActionActionStop, InstanceId: types.StringPtr("2")}).
		Return(core.InstanceActionResponse{}, nil)

	err := p.StopInstance(ctx, "instance-2")

	assert.NilError(t, err)
}

func TestAddInstancesNetworkDetails(t *testing.T) {
	instances := &[]lepton.CloudInstance{
		{ID: "1", Status: "RUNNING", Name: "instance-1", Created: "a long while ago"},
		{ID: "2", Status: "STOPPED", Name: "instance-2", Created: "a long while ago"},
	}

	p, c, _, _, n, _, _ := NewProvider(t)

	c.EXPECT().
		ListVnicAttachments(context.TODO(), core.ListVnicAttachmentsRequest{CompartmentId: types.StringPtr(""), InstanceId: types.StringPtr("1")}).
		Return(core.ListVnicAttachmentsResponse{
			Items: []core.VnicAttachment{
				{VnicId: types.StringPtr("1")},
				{VnicId: types.StringPtr("2")},
			},
		}, nil)

	c.EXPECT().
		ListVnicAttachments(context.TODO(), core.ListVnicAttachmentsRequest{CompartmentId: types.StringPtr(""), InstanceId: types.StringPtr("2")}).
		Return(core.ListVnicAttachmentsResponse{
			Items: []core.VnicAttachment{},
		}, nil)

	n.EXPECT().
		GetVnic(context.TODO(), core.GetVnicRequest{VnicId: types.StringPtr("1")}).
		Return(core.GetVnicResponse{
			Vnic: core.Vnic{
				PrivateIp: types.StringPtr("10.10.10.10"),
				PublicIp:  types.StringPtr("192.168.1.1"),
			},
		}, nil)

	n.EXPECT().
		GetVnic(context.TODO(), core.GetVnicRequest{VnicId: types.StringPtr("2")}).
		Return(core.GetVnicResponse{
			Vnic: core.Vnic{
				PrivateIp: types.StringPtr("10.10.10.20"),
				PublicIp:  types.StringPtr("192.168.1.2"),
			},
		}, nil)

	ctx := lepton.NewContext(lepton.NewConfig())

	err := p.AddInstancesNetworkDetails(ctx, instances)

	assert.NilError(t, err)

	expected := &[]lepton.CloudInstance{
		{ID: "1", Status: "RUNNING", Name: "instance-1", Created: "a long while ago", PublicIps: []string{"192.168.1.1", "192.168.1.2"}, PrivateIps: []string{"10.10.10.10", "10.10.10.20"}},
		{ID: "2", Status: "STOPPED", Name: "instance-2", Created: "a long while ago"},
	}

	assert.DeepEqual(t, instances, expected)

}

func defaultInstancesList() core.ListInstancesResponse {
	return core.ListInstancesResponse{
		Items: []core.Instance{
			{Id: types.StringPtr("1"), LifecycleState: core.InstanceLifecycleStateRunning, DisplayName: types.StringPtr("instance-1"), TimeCreated: &common.SDKTime{Time: time.Unix(1000, 0)}, FreeformTags: ociOpsTags},
			{Id: types.StringPtr("2"), LifecycleState: core.InstanceLifecycleStateStopped, DisplayName: types.StringPtr("instance-2"), TimeCreated: &common.SDKTime{Time: time.Unix(1000, 0)}, FreeformTags: ociOpsTags},
			{Id: types.StringPtr("3"), LifecycleState: core.InstanceLifecycleStateTerminated, DisplayName: types.StringPtr("instance-3"), TimeCreated: &common.SDKTime{Time: time.Unix(1000, 0)}, FreeformTags: ociOpsTags},
		},
	}
}
