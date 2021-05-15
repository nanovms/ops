package oci_test

import (
	"context"
	"testing"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/testutils"
	"github.com/nanovms/ops/types"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"gotest.tools/assert"
)

func TestCreateInstance(t *testing.T) {
	p, c, _, _, n, _, _ := NewProvider(t)
	ctx := testutils.NewMockContext()
	instanceName := "instance-test"
	imageName := "image-test"

	ctx.Config().RunConfig.InstanceName = instanceName
	ctx.Config().CloudConfig.ImageName = imageName
	ctx.Config().RunConfig.Ports = []string{"80"}
	ctx.Config().RunConfig.UDPPorts = []string{"800-1000"}

	privateSubnet := core.Subnet{
		Id:                     types.StringPtr("private-subnet"),
		ProhibitPublicIpOnVnic: types.BoolPtr(true),
		VcnId:                  types.StringPtr("vcn-1"),
	}

	publicSubnet := core.Subnet{
		Id:                     types.StringPtr("public-subnet"),
		ProhibitPublicIpOnVnic: types.BoolPtr(false),
		VcnId:                  types.StringPtr("vcn-1"),
	}

	n.EXPECT().
		ListSubnets(context.TODO(), core.ListSubnetsRequest{CompartmentId: types.StringPtr("")}).
		Return(core.ListSubnetsResponse{Items: []core.Subnet{
			privateSubnet,
			publicSubnet,
		}}, nil)

	n.EXPECT().
		CreateNetworkSecurityGroup(context.TODO(), core.CreateNetworkSecurityGroupRequest{
			CreateNetworkSecurityGroupDetails: core.CreateNetworkSecurityGroupDetails{
				CompartmentId: types.StringPtr(""),
				VcnId:         types.StringPtr("vcn-1"),
				DisplayName:   types.StringPtr(instanceName + "-sg"),
			},
		}).
		Return(core.CreateNetworkSecurityGroupResponse{
			NetworkSecurityGroup: core.NetworkSecurityGroup{
				Id: types.StringPtr("instance-sg-1"),
			},
		}, nil)

	sgRules := []core.AddSecurityRuleDetails{
		{
			Direction:       core.AddSecurityRuleDetailsDirectionIngress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Source:          types.StringPtr("0.0.0.0/0"),
			Protocol:        types.StringPtr("6"),
			TcpOptions: &core.TcpOptions{
				SourcePortRange: &core.PortRange{
					Min: types.IntPtr(80),
					Max: types.IntPtr(80),
				},
			},
		},
		{
			Direction:       core.AddSecurityRuleDetailsDirectionEgress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Destination:     types.StringPtr("0.0.0.0/0"),
			Protocol:        types.StringPtr("6"),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: types.IntPtr(80),
					Max: types.IntPtr(80),
				},
			},
		},
		{
			Direction:       core.AddSecurityRuleDetailsDirectionIngress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Source:          types.StringPtr("0.0.0.0/0"),
			Protocol:        types.StringPtr("17"),
			UdpOptions: &core.UdpOptions{
				SourcePortRange: &core.PortRange{
					Min: types.IntPtr(800),
					Max: types.IntPtr(1000),
				},
			},
		},
		{
			Direction:       core.AddSecurityRuleDetailsDirectionEgress,
			DestinationType: core.AddSecurityRuleDetailsDestinationTypeCidrBlock,
			Destination:     types.StringPtr("0.0.0.0/0"),
			Protocol:        types.StringPtr("17"),
			UdpOptions: &core.UdpOptions{
				DestinationPortRange: &core.PortRange{
					Min: types.IntPtr(800),
					Max: types.IntPtr(1000),
				},
			},
		},
	}

	n.EXPECT().
		AddNetworkSecurityGroupSecurityRules(context.TODO(), core.AddNetworkSecurityGroupSecurityRulesRequest{
			NetworkSecurityGroupId: types.StringPtr("instance-sg-1"),
			AddNetworkSecurityGroupSecurityRulesDetails: core.AddNetworkSecurityGroupSecurityRulesDetails{
				SecurityRules: sgRules,
			}})

	c.EXPECT().
		ListImages(context.TODO(), core.ListImagesRequest{CompartmentId: types.StringPtr("")}).
		Return(core.ListImagesResponse{
			Items: []core.Image{
				{Id: types.StringPtr("1"), DisplayName: &imageName, TimeCreated: &common.SDKTime{Time: time.Unix(1000, 0)}, SizeInMBs: types.Int64Ptr(100000), FreeformTags: ociOpsTags},
			},
		}, nil)

	c.EXPECT().
		LaunchInstance(context.TODO(), core.LaunchInstanceRequest{
			LaunchInstanceDetails: core.LaunchInstanceDetails{
				AvailabilityDomain: types.StringPtr(""),
				CompartmentId:      types.StringPtr(""),
				CreateVnicDetails: &core.CreateVnicDetails{
					SubnetId: publicSubnet.Id,
					NsgIds:   []string{"instance-sg-1"},
				},
				DisplayName: &instanceName,
				Shape:       types.StringPtr("VM.Standard2.1"),
				SourceDetails: core.InstanceSourceViaImageDetails{
					ImageId: types.StringPtr("1"),
				},
				FreeformTags: createTags(map[string]string{"Image": "image-test"}),
			},
		})

	err := p.CreateInstance(ctx)

	assert.NilError(t, err)
}

func TestGetInstances(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := testutils.NewMockContext()

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
	ctx := testutils.NewMockContext()

	c.EXPECT().
		ListInstances(context.TODO(), core.ListInstancesRequest{CompartmentId: types.StringPtr("")}).
		Return(defaultInstancesList(), nil)

	instance, err := p.GetInstance(ctx, "instance-2")

	assert.NilError(t, err)

	expected := &lepton.CloudInstance{ID: "2", Status: "STOPPED", Name: "instance-2", Created: "a long while ago"}

	assert.DeepEqual(t, expected, instance)
}

func TestDeleteInstance(t *testing.T) {
	p, c, _, _, _, _, _ := NewProvider(t)
	ctx := testutils.NewMockContext()

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
	ctx := testutils.NewMockContext()

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
	ctx := testutils.NewMockContext()

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

	ctx := testutils.NewMockContext()

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
