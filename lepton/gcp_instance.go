package lepton

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// CreateInstance - Creates instance on Google Cloud Platform
func (p *GCloud) CreateInstance(ctx *Context) error {
	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return err
	}

	c := ctx.config
	if c.CloudConfig.Zone == "" {
		return fmt.Errorf("Zone not provided in config.CloudConfig")
	}

	if c.CloudConfig.ProjectID == "" {
		fmt.Printf("ProjectId not provided in config.CloudConfig. Using %s from default credentials.\n", creds.ProjectID)
		c.CloudConfig.ProjectID = creds.ProjectID
	}

	if c.CloudConfig.Flavor == "" {
		c.CloudConfig.Flavor = "g1-small"
	}

	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	nic, err := p.getNIC(ctx, computeService)
	if err != nil {
		return err
	}

	machineType := fmt.Sprintf("zones/%s/machineTypes/%s", c.CloudConfig.Zone, c.CloudConfig.Flavor)

	imageName := fmt.Sprintf("projects/%v/global/images/%v",
		c.CloudConfig.ProjectID,
		c.CloudConfig.ImageName)

	serialTrue := "true"

	labels := map[string]string{}
	for _, tag := range ctx.config.RunConfig.Tags {
		labels[tag.Key] = tag.Value
	}

	instanceName := c.RunConfig.InstanceName

	rb := &compute.Instance{
		Name:        instanceName,
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
		Labels: buildGcpTags(ctx.config.RunConfig.Tags),
		Tags: &compute.Tags{
			Items: []string{instanceName},
		},
	}
	op, err := computeService.Instances.Insert(c.CloudConfig.ProjectID, c.CloudConfig.Zone, rb).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation started using image %s. Monitoring operation %s.\n", imageName, op.Name)
	err = p.pollOperation(context, c.CloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation succeeded %s.\n", instanceName)

	// create dns zones/records to associate DNS record to instance IP
	if c.RunConfig.DomainName != "" {
		instance, err := computeService.Instances.Get(c.CloudConfig.ProjectID, c.CloudConfig.Zone, instanceName).Do()
		if err != nil {
			return err
		}

		cinstance := p.convertToCloudInstance(instance)

		if len(cinstance.PublicIps) != 0 {
			ctx.logger.Info("Assigning IP %s to %s", cinstance.PublicIps[0], c.RunConfig.DomainName)
			err := CreateDNSRecord(ctx.config, cinstance.PublicIps[0], p)
			if err != nil {
				return err
			}
		}
	}

	// create firewall rules to expose instance ports
	if len(ctx.config.RunConfig.Ports) != 0 {
		rule := p.buildFirewallRule("tcp", ctx.config.RunConfig.Ports, instanceName)

		_, err = computeService.Firewalls.Insert(c.CloudConfig.ProjectID, rule).Context(context).Do()

		if err != nil {
			ctx.logger.Error("%v", err)
			return errors.New("Failed to add Firewall rule")
		}
	}

	if len(ctx.config.RunConfig.UDPPorts) != 0 {
		rule := p.buildFirewallRule("udp", ctx.config.RunConfig.UDPPorts, instanceName)

		_, err = computeService.Firewalls.Insert(c.CloudConfig.ProjectID, rule).Context(context).Do()

		if err != nil {
			ctx.logger.Error("%v", err)
			return errors.New("Failed to add Firewall rule")
		}
	}

	return nil
}

// ListInstances lists instances on Gcloud
func (p *GCloud) ListInstances(ctx *Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range instances {
		var rows []string
		rows = append(rows, instance.Name)
		rows = append(rows, instance.Status)
		rows = append(rows, instance.Created)
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))
		table.Append(rows)
	}
	table.Render()
	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (p *GCloud) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	req := p.Service.Instances.Get(ctx.config.CloudConfig.ProjectID, ctx.config.CloudConfig.Zone, id)

	instance, err := req.Do()
	if err != nil {
		return nil, err
	}

	return p.convertToCloudInstance(instance), nil
}

// GetInstances return all instances on GCloud
func (p *GCloud) GetInstances(ctx *Context) ([]CloudInstance, error) {
	context := context.TODO()
	var (
		cinstances []CloudInstance
		req        = p.Service.Instances.List(ctx.config.CloudConfig.ProjectID, ctx.config.CloudConfig.Zone)
	)

	if err := req.Pages(context, func(page *compute.InstanceList) error {
		for _, instance := range page.Items {
			if val, ok := instance.Labels["createdby"]; ok && val == "ops" {
				cinstance := p.convertToCloudInstance(instance)
				cinstances = append(cinstances, *cinstance)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return cinstances, nil
}

func (p *GCloud) convertToCloudInstance(instance *compute.Instance) *CloudInstance {
	var (
		privateIps, publicIps []string
	)
	for _, ninterface := range instance.NetworkInterfaces {
		if ninterface.NetworkIP != "" {
			privateIps = append(privateIps, ninterface.NetworkIP)

		}
		for _, accessConfig := range ninterface.AccessConfigs {
			if accessConfig.NatIP != "" {
				publicIps = append(publicIps, accessConfig.NatIP)
			}
		}
	}

	return &CloudInstance{
		Name:       instance.Name,
		Status:     instance.Status,
		Created:    instance.CreationTimestamp,
		PublicIps:  publicIps,
		PrivateIps: privateIps,
	}
}

// DeleteInstance deletes instance from Gcloud
func (p *GCloud) DeleteInstance(ctx *Context, instancename string) error {
	context := context.TODO()
	cloudConfig := ctx.config.CloudConfig
	op, err := p.Service.Instances.Delete(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance deletion started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Instance deletion succeeded %s.\n", instancename)
	return nil
}

// StartInstance starts an instance in GCloud
func (p *GCloud) StartInstance(ctx *Context, instancename string) error {

	context := context.TODO()

	cloudConfig := ctx.config.CloudConfig
	op, err := p.Service.Instances.Start(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}

	fmt.Printf("Instance started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Instance started %s.\n", instancename)
	return nil

}

// StopInstance stops instance
func (p *GCloud) StopInstance(ctx *Context, instancename string) error {
	context := context.TODO()

	cloudConfig := ctx.config.CloudConfig
	op, err := p.Service.Instances.Stop(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}

	fmt.Printf("Instance stopping started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Instance stop succeeded %s.\n", instancename)
	return nil
}

// ResetInstance resets instance
func (p *GCloud) ResetInstance(ctx *Context, instancename string) error {
	context := context.TODO()

	cloudConfig := ctx.config.CloudConfig
	op, err := p.Service.Instances.Reset(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}

	fmt.Printf("Instance reseting started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, p.Service, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Instance reseting succeeded %s.\n", instancename)
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *GCloud) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (p *GCloud) GetInstanceLogs(ctx *Context, instancename string) (string, error) {
	context := context.TODO()

	cloudConfig := ctx.config.CloudConfig
	lastPos := int64(0)

	resp, err := p.Service.Instances.GetSerialPortOutput(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Start(lastPos).Context(context).Do()
	if err != nil {
		return "", err
	}
	if resp.Contents != "" {
		return resp.Contents, nil
	}

	lastPos = resp.Next
	time.Sleep(time.Second)

	return "", nil
}
