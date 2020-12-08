package lepton

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance - Creates instance on Digital Ocean Platform
func (do *DigitalOcean) CreateInstance(ctx *Context) error {
	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (do *DigitalOcean) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// GetInstances return all instances on DigitalOcean
// TODO
func (do *DigitalOcean) GetInstances(ctx *Context) ([]CloudInstance, error) {
	opt := &godo.ListOptions{}
	list, _, err := do.Client.Droplets.List(context.TODO(), opt)
	if err != nil {
		return nil, err
	}
	cinstances := make([]CloudInstance, len(list))
	for i, droplet := range list {
		privateIPV4, _ := droplet.PrivateIPv4()
		publicIPV4, _ := droplet.PublicIPv4()
		publicIPV6, _ := droplet.PublicIPv6()
		cinstances[i] = CloudInstance{
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
func (do *DigitalOcean) ListInstances(ctx *Context) error {
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
func (do *DigitalOcean) DeleteInstance(ctx *Context, instancename string) error {
	return nil
}

// StartInstance starts an instance in DO
func (do *DigitalOcean) StartInstance(ctx *Context, instancename string) error {
	return nil
}

// StopInstance deletes instance from DO
func (do *DigitalOcean) StopInstance(ctx *Context, instancename string) error {
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (do *DigitalOcean) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := do.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}
