package gcp

import (
	"context"
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	compute "google.golang.org/api/compute/v1"
)

func (p *GCloud) createInstanceTemplate(ctx *lepton.Context, instanceGroup string) (string, error) {
	c := ctx.Config()

	instanceName := c.RunConfig.InstanceName
	machineType := c.CloudConfig.Flavor

	imageName := fmt.Sprintf("projects/%v/global/images/%v",
		c.CloudConfig.ProjectID,
		c.CloudConfig.ImageName)

	serialTrue := "true"

	nic, err := p.getNIC(ctx, p.Service)
	if err != nil {
		return "", err
	}

	it := &compute.InstanceTemplate{
		Name: instanceName,
		Properties: &compute.InstanceProperties{
			MachineType: machineType,
			Disks: []*compute.AttachedDisk{
				{
					AutoDelete: true,
					Boot:       true,
					Type:       "PERSISTENT",
					InitializeParams: &compute.AttachedDiskInitializeParams{
						SourceImage: imageName,
					},
				},
			},
			NetworkInterfaces: nic,
			Metadata: &compute.Metadata{
				Items: []*compute.MetadataItems{
					{
						Key:   "serial-port-enable",
						Value: &serialTrue,
					},
				},
			},
			Labels: buildGcpTags(ctx.Config().CloudConfig.Tags),
			Tags: &compute.Tags{
				Items: []string{instanceName},
			},
		},
	}

	op, err := p.Service.InstanceTemplates.Insert(c.CloudConfig.ProjectID, it).Context(context.TODO()).Do()
	if err != nil {
		return "", err
	}
	fmt.Printf("Instance template creation started.")

	err = p.pollOperation(context.TODO(), c.CloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return "", err
	}

	// add network tags for ports

	if len(ctx.Config().RunConfig.Ports) != 0 {
		rule := p.buildFirewallRule("tcp", ctx.Config().RunConfig.Ports, instanceName, ctx.Config().CloudConfig.Subnet, false)

		_, err = p.Service.Firewalls.Insert(c.CloudConfig.ProjectID, rule).Context(context.TODO()).Do()

		if err != nil {
			return "", err
		}
	}

	if len(ctx.Config().RunConfig.UDPPorts) != 0 {
		rule := p.buildFirewallRule("udp", ctx.Config().RunConfig.UDPPorts, instanceName, ctx.Config().CloudConfig.Subnet, false)

		_, err = p.Service.Firewalls.Insert(c.CloudConfig.ProjectID, rule).Context(context.TODO()).Do()

		if err != nil {
			return "", err
		}
	}

	return op.SelfLink, nil
}

func (p *GCloud) replaceInstanceTemplate(ctx *lepton.Context, instanceGroup string, iturl string) error {
	c := ctx.Config()

	itName := c.RunConfig.InstanceName

	it, err := p.Service.InstanceTemplates.Get(c.CloudConfig.ProjectID, itName).Do()
	if err != nil {
		ctx.Logger().Errorf("failed getting instance")
		return err
	}

	tmp := &compute.InstanceGroupManagersSetInstanceTemplateRequest{
		InstanceTemplate: it.SelfLink,
	}

	op, err := p.Service.InstanceGroupManagers.SetInstanceTemplate(c.CloudConfig.ProjectID, c.CloudConfig.Zone, instanceGroup, tmp).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	fmt.Printf("replacing instance template in instance group")

	err = p.pollOperation(context.TODO(), c.CloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("done updating instance group to use new instance template")

	return nil
}

func (p *GCloud) updateInstanceGroup(ctx *lepton.Context, instanceGroup string) error {
	c := ctx.Config()

	managedInstances, err := p.Service.InstanceGroupManagers.ListManagedInstances(
		c.CloudConfig.ProjectID, c.CloudConfig.Zone, instanceGroup).Do()

	managedInstanceCount := len(managedInstances.ManagedInstances)
	instances := make([]string, managedInstanceCount)
	for i, v := range managedInstances.ManagedInstances {
		instances[i] = v.Instance
	}

	tmp := &compute.InstanceGroupManagersRecreateInstancesRequest{
		Instances: instances,
	}

	op, err := p.Service.InstanceGroupManagers.RecreateInstances(c.CloudConfig.ProjectID, c.CloudConfig.Zone, instanceGroup, tmp).Context(context.TODO()).Do()
	if err != nil {
		return err
	}

	err = p.pollOperation(context.TODO(), c.CloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("updating instance group to immediately re-create instances")

	return nil
}

// mv me to gcp_instance_group
func (p *GCloud) addToInstanceGroup(ctx *lepton.Context, instanceGroup string) {
	iturl, err := p.createInstanceTemplate(ctx, instanceGroup)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = p.replaceInstanceTemplate(ctx, instanceGroup, iturl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = p.updateInstanceGroup(ctx, instanceGroup)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
