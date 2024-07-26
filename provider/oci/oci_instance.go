//go:build oci || !onlyprovider

package oci

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"github.com/oracle/oci-go-sdk/core"
)

// CreateInstance launch a server in oci using an existing image
func (p *Provider) CreateInstance(ctx *lepton.Context) error {
	instanceName := ctx.Config().RunConfig.InstanceName
	securityGroups := []string{}

	image, err := p.getImageByName(ctx, ctx.Config().CloudConfig.ImageName)
	if err != nil {
		return err
	}

	flavor := ctx.Config().CloudConfig.Flavor
	if flavor == "" {
		flavor = "VM.Standard2.1"
	}

	subnet, err := p.GetSubnet(ctx)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed getting subnet")
	}

	sg, err := p.CreateNetworkSecurityGroup(ctx, *subnet.VcnId)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed creating network security group")
	}

	securityGroups = append(securityGroups, *sg.Id)

	tags := map[string]string{}
	for _, tag := range ctx.Config().CloudConfig.Tags {
		tags[tag.Key] = tag.Value
	}
	for k, v := range ociOpsTags {
		tags[k] = v
	}
	tags["Image"] = image.Name

	_, err = p.computeClient.LaunchInstance(context.TODO(), core.LaunchInstanceRequest{
		LaunchInstanceDetails: core.LaunchInstanceDetails{
			AvailabilityDomain: types.StringPtr(p.availabilityDomain),
			CompartmentId:      types.StringPtr(p.compartmentID),
			CreateVnicDetails: &core.CreateVnicDetails{
				SubnetId: subnet.Id,
				NsgIds:   securityGroups,
			},
			DisplayName: types.StringPtr(instanceName),
			Shape:       types.StringPtr(flavor),
			SourceDetails: core.InstanceSourceViaImageDetails{
				ImageId: &image.ID,
			},
			FreeformTags: tags,
		},
	})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed launching instance")
	}

	return nil
}

// ListInstances prints servers list managed by oci in table
func (p *Provider) ListInstances(ctx *lepton.Context) (err error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return
	}

	err = p.AddInstancesNetworkDetails(ctx, &instances)
	if err != nil {
		ctx.Logger().Error(err)
		err = errors.New("failed getting instances network details")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Private Ips", "Public Ips", "Image", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

	table.SetRowLine(true)

	for _, i := range instances {
		var rows []string

		rows = append(rows, i.Name)
		rows = append(rows, i.Status)
		rows = append(rows, strings.Join(i.PrivateIps, ", "))
		rows = append(rows, strings.Join(i.PublicIps, ", "))
		rows = append(rows, i.Image)
		rows = append(rows, i.Created)

		table.Append(rows)
	}

	table.Render()

	return
}

// AddInstancesNetworkDetails append public IPs and private IPs to instances passed in argumetn
func (p *Provider) AddInstancesNetworkDetails(ctx *lepton.Context, instances *[]lepton.CloudInstance) (err error) {
	instancesN := len(*instances)

	errorsMessages := ""
	responses := make(chan error, instancesN)

	addInstanceNetworkDetails := func(i int) {
		instance := (*instances)[i]
		vnicsAttachments, err := p.computeClient.ListVnicAttachments(context.TODO(), core.ListVnicAttachmentsRequest{CompartmentId: &p.compartmentID, InstanceId: &instance.ID})
		if err != nil {
			ctx.Logger().Error(err)
			responses <- errors.New("failed getting vnic attachments of instance " + instance.ID + "\n")
			return
		}

		for _, vnic := range vnicsAttachments.Items {
			vnicDetails, err := p.networkClient.GetVnic(context.TODO(), core.GetVnicRequest{VnicId: vnic.VnicId})
			if err != nil {
				ctx.Logger().Error(err)
				responses <- errors.New("failed getting details of vnic " + *vnic.VnicId + "\n")
				return
			}

			(*instances)[i].PublicIps = append((*instances)[i].PublicIps, *vnicDetails.Vnic.PublicIp)
			(*instances)[i].PrivateIps = append((*instances)[i].PrivateIps, *vnicDetails.Vnic.PrivateIp)
		}

		responses <- nil
	}

	for index := range *instances {
		go addInstanceNetworkDetails(index)
	}

	for i := 0; i < instancesN; i++ {
		err := <-responses

		if err != nil {
			errorsMessages += err.Error() + "\n"
		}
	}

	if len(errorsMessages) > 0 {
		err = errors.New(errorsMessages)
	}

	return
}

// GetInstances returns the list of servers managed by upcloud
func (p *Provider) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	instances = []lepton.CloudInstance{}

	result, err := p.computeClient.ListInstances(context.TODO(), core.ListInstancesRequest{
		CompartmentId: types.StringPtr(p.compartmentID),
	})
	if err != nil {
		return
	}

	for _, i := range result.Items {
		if i.LifecycleState != core.InstanceLifecycleStateTerminated && checkHasOpsTags(i.FreeformTags) {
			instances = append(instances, lepton.CloudInstance{
				ID:      *i.Id,
				Name:    *i.DisplayName,
				Status:  string(i.LifecycleState),
				Created: lepton.Time2Human(i.TimeCreated.Time),
				Image:   i.FreeformTags["Image"],
			})
		}
	}

	return
}

// GetInstanceByName returns oci instance with given name
func (p *Provider) GetInstanceByName(ctx *lepton.Context, name string) (instance *lepton.CloudInstance, err error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return
	}

	for _, i := range instances {
		if i.Name == name {
			instance = &i
			return
		}
	}

	err = lepton.ErrInstanceNotFound(name)
	return
}

// DeleteInstance removes an instance
func (p *Provider) DeleteInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetInstanceByName(ctx, instancename)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed getting instance")
	}

	_, err = p.computeClient.TerminateInstance(context.TODO(), core.TerminateInstanceRequest{InstanceId: &instance.ID})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed terminate instance")
	}

	return nil
}

// StopInstance stops an instance
func (p *Provider) StopInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetInstanceByName(ctx, instancename)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed getting instance")
	}

	_, err = p.computeClient.InstanceAction(context.TODO(), core.InstanceActionRequest{Action: core.InstanceActionActionStop, InstanceId: &instance.ID})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed terminate instance")
	}

	return nil
}

// RebootInstance reboots the instance.
func (p *Provider) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance
func (p *Provider) StartInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetInstanceByName(ctx, instancename)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed getting instance")
	}

	_, err = p.computeClient.InstanceAction(context.TODO(), core.InstanceActionRequest{Action: core.InstanceActionActionStart, InstanceId: &instance.ID})
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed terminate instance")
	}

	return nil
}

// GetInstanceLogs returns instance log
func (p *Provider) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// PrintInstanceLogs prints instances logs on console
func (p *Provider) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// InstanceStats show metrics for instances on provider.
func (p *Provider) InstanceStats(ctx *lepton.Context, instancename string) error {
	return errors.New("currently not avilable")
}
