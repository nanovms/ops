package digitalocean

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

const (
	fakeFingerprint = "02:14:c3:d2:cd:4a:67:8b:43:b3:49:83:99:49:6a:58"
)

// CreateInstance - Creates instance on Digital Ocean Platform
func (do *DigitalOcean) CreateInstance(ctx *lepton.Context) error {
	config := ctx.Config()

	image, err := do.getImageByName(ctx, config.CloudConfig.ImageName)
	if err != nil {
		return err
	}

	imageID, _ := strconv.Atoi(image.ID)

	flavor := "s-1vcpu-1gb"
	if config.CloudConfig.Flavor != "" {
		flavor = config.CloudConfig.Flavor
	}

	createReq := &godo.DropletCreateRequest{
		Name:   config.RunConfig.InstanceName,
		Size:   flavor,
		Region: config.CloudConfig.Zone,
		Image: godo.DropletCreateImage{
			ID: imageID,
		},
		Tags:    []string{opsTag},
		SSHKeys: []godo.DropletCreateSSHKey{{Fingerprint: fakeFingerprint}},
	}

	_, _, err = do.Client.Droplets.Create(context.TODO(), createReq)
	if err != nil {
		return err
	}

	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (do *DigitalOcean) GetInstanceByID(ctx *lepton.Context, id string) (droplet *lepton.CloudInstance, err error) {
	droplets, err := do.GetInstances(ctx)
	if err != nil {
		return
	}

	for _, d := range droplets {
		if d.Name == id {
			droplet = &d
		}
	}

	if droplet == nil {
		err = fmt.Errorf(`droplet with name "%s" not found`, id)
		return
	}

	return
}

// GetInstances return all instances on DigitalOcean
// TODO
func (do *DigitalOcean) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Droplets.ListByTag(context.TODO(), opsTag, opt)
	if err != nil {
		return nil, err
	}
	cinstances := make([]lepton.CloudInstance, len(list))
	for i, droplet := range list {
		privateIPV4, _ := droplet.PrivateIPv4()
		publicIPV4, _ := droplet.PublicIPv4()
		publicIPV6, _ := droplet.PublicIPv6()
		cinstances[i] = lepton.CloudInstance{
			ID:         fmt.Sprintf("%d", droplet.ID),
			Name:       droplet.Name,
			Status:     droplet.Status,
			Created:    droplet.Created,
			PrivateIps: []string{privateIPV4},
			PublicIps:  []string{publicIPV4, publicIPV6},
		}
	}

	return cinstances, nil
}

// ListInstances lists instances on DO
func (do *DigitalOcean) ListInstances(ctx *lepton.Context) error {
	instances, err := do.GetInstances(ctx)
	if err != nil {
		return err
	}
	// print list of images in table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, instance := range instances {
		var rows []string
		rows = append(rows, instance.Name)
		rows = append(rows, instance.ID)
		rows = append(rows, instance.Status)
		rows = append(rows, instance.Created)
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))

		table.Append(rows)
	}
	table.Render()
	return nil

}

// DeleteInstance deletes instance from DO
func (do *DigitalOcean) DeleteInstance(ctx *lepton.Context, instancename string) error {
	instance, err := do.GetInstanceByID(ctx, instancename)
	if err != nil {
		return err
	}

	instanceID, _ := strconv.Atoi(instance.ID)

	_, err = do.Client.Droplets.Delete(context.TODO(), instanceID)
	if err != nil {
		return err
	}

	return nil
}

// StartInstance starts an instance in DO
func (do *DigitalOcean) StartInstance(ctx *lepton.Context, instancename string) error {
	instance, err := do.GetInstanceByID(ctx, instancename)
	if err != nil {
		return err
	}

	instanceID, _ := strconv.Atoi(instance.ID)

	_, _, err = do.Client.DropletActions.PowerOn(context.TODO(), instanceID)
	if err != nil {
		return err
	}

	return nil
}

// StopInstance deletes instance from DO
func (do *DigitalOcean) StopInstance(ctx *lepton.Context, instancename string) error {
	instance, err := do.GetInstanceByID(ctx, instancename)
	if err != nil {
		return err
	}

	instanceID, _ := strconv.Atoi(instance.ID)

	_, _, err = do.Client.DropletActions.PowerOff(context.TODO(), instanceID)
	if err != nil {
		return err
	}

	return nil
}

// GetInstanceLogs gets instance related logs
func (do *DigitalOcean) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// PrintInstanceLogs writes instance logs to console
func (do *DigitalOcean) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := do.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}
