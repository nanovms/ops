//go:build digitalocean || do || !onlyprovider

package digitalocean

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
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

	instanceName := config.RunConfig.InstanceName
	imageName := "image:" + image.Name

	keys, _, err := do.Client.Keys.List(context.TODO(), &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	})
	if err != nil {
		return err
	}

	dropletKeys := []godo.DropletCreateSSHKey{}

	for _, key := range keys {
		dropletKeys = append(dropletKeys, godo.DropletCreateSSHKey{
			ID:          key.ID,
			Fingerprint: key.Fingerprint,
		})
	}

	tags := []string{opsTag, imageName}
	for _, t := range config.CloudConfig.Tags {
		// NOTE: this would allow for tags without : in them
		if t.Key == "" && t.Value != "" {
			tags = append(tags, t.Value)
		} else if t.Key != "" && t.Value != "" {
			tags = append(tags, fmt.Sprintf("%s:%s", t.Key, t.Value))
		}
	}

	// if users set VPC uuid instead of name it avoids lookup
	vpcUUID := config.CloudConfig.VPC
	if vpcUUID != "" {
		if _, err := uuid.Parse(vpcUUID); err != nil {
			vpc, err := do.GetVPC(ctx, vpcUUID)
			if err != nil {
				return err
			}
			if vpc != nil {
				vpcUUID = vpc.ID
			}
		}
	}

	createReq := &godo.DropletCreateRequest{
		Name:   instanceName,
		Size:   flavor,
		Region: config.CloudConfig.Zone,
		Image: godo.DropletCreateImage{
			ID: imageID,
		},
		Tags:    tags,
		SSHKeys: dropletKeys,
	}
	if vpcUUID != "" {
		createReq.VPCUUID = vpcUUID
	}

	_, _, err = do.Client.Droplets.Create(context.TODO(), createReq)
	if err != nil {
		return err
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (do *DigitalOcean) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	droplets, err := do.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	if len(droplets) == 0 {
		return nil, lepton.ErrInstanceNotFound(name)
	}

	for _, d := range droplets {
		if d.Name == name {
			return &d, nil
		}
	}

	return nil, lepton.ErrInstanceNotFound(name)
}

// GetInstances return all instances on DigitalOcean
// TODO
func (do *DigitalOcean) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Droplets.ListByTag(context.TODO(), opsTag, opt)
	if err != nil {
		return nil, err
	}

	cinstances := []lepton.CloudInstance{}
	for _, droplet := range list {
		privateIPV4, _ := droplet.PrivateIPv4()
		publicIPV4, _ := droplet.PublicIPv4()
		publicIPV6, _ := droplet.PublicIPv6()

		instance := lepton.CloudInstance{
			ID:         fmt.Sprintf("%d", droplet.ID),
			Name:       droplet.Name,
			Status:     droplet.Status,
			Created:    droplet.Created,
			PrivateIps: []string{privateIPV4},
			PublicIps:  []string{publicIPV4, publicIPV6},
		}

		isOpsImage := false

		for _, t := range droplet.Tags {
			if t == opsTag {
				isOpsImage = true
			} else if strings.Contains(t, "image:") {
				parts := strings.Split(t, ":")
				if len(parts) > 1 {
					instance.Image = parts[1]
				}
			}
		}

		if isOpsImage {
			cinstances = append(cinstances, instance)
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
	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(instances)
	}
	// print list of images in table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Id", "Status", "Created", "Private Ips", "Public Ips", "Image"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
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
		rows = append(rows, instance.Image)
		table.Append(rows)
	}
	table.Render()
	return nil

}

// DeleteInstance deletes instance from DO
func (do *DigitalOcean) DeleteInstance(ctx *lepton.Context, instancename string) error {
	instance, err := do.GetInstanceByName(ctx, instancename)
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

// RebootInstance reboots the instance.
func (do *DigitalOcean) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance in DO
func (do *DigitalOcean) StartInstance(ctx *lepton.Context, instancename string) error {
	instance, err := do.GetInstanceByName(ctx, instancename)
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
	instance, err := do.GetInstanceByName(ctx, instancename)
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

// InstanceStats show metrics for instances on digitalocean.
func (do *DigitalOcean) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not avilable")
}

// PrintInstanceLogs writes instance logs to console
func (do *DigitalOcean) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := do.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf("%s", l)
	return nil
}
