package vultr

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
	"github.com/vultr/govultr/v2"
)

// CreateInstance - Creates instance on Digital Ocean Platform
func (v *Vultr) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	flavor := "vc2-1c-1gb"
	if c.CloudConfig.Flavor != "" {
		flavor = c.CloudConfig.Flavor
	}

	if c.CloudConfig.Zone == "" {
		return fmt.Errorf("zone is required (-z)")
	}

	instance, err := v.Client.Instance.Create(context.TODO(), &govultr.InstanceCreateReq{
		Region:     c.CloudConfig.Zone,
		Plan:       flavor,
		SnapshotID: c.CloudConfig.ImageName,
	})
	if err != nil {
		return err
	}

	log.Info("instance:", instance)

	return nil
}

// GetInstanceByName returns instance with given name
func (v *Vultr) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	instance, err := v.Client.Instance.Get(context.TODO(), name)
	if err != nil {
		return nil, err
	}

	return &lepton.CloudInstance{
		ID:        instance.ID,
		Status:    instance.Status,
		Created:   time.Now().String(),
		PublicIps: []string{instance.MainIP},
		Ports:     []string{},
		Image:     instance.ImageID,
	}, nil
}

// GetInstances return all instances on Vultr
func (v *Vultr) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	instances, _, err := v.Client.Instance.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
	})
	if err != nil {
		return nil, err
	}

	var cloudInstances []lepton.CloudInstance

	for _, instance := range instances {
		cloudInstances = append(cloudInstances, lepton.CloudInstance{
			ID:        instance.ID,
			Status:    instance.Status,
			Created:   time.Now().String(),
			PublicIps: []string{instance.MainIP},
			Image:     instance.ImageID,
		})
	}

	return cloudInstances, nil

}

// ListInstances lists instances on v
func (v *Vultr) ListInstances(ctx *lepton.Context) error {

	instances, _, err := v.Client.Instance.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
	})
	if err != nil {
		return err
	}

	log.Debug("instances:", instances)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Plan", "MainIP", "Status", "ImageID", "Region"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range instances {
		var row []string
		row = append(row, instance.ID)
		row = append(row, instance.Plan)
		row = append(row, instance.MainIP)
		row = append(row, instance.Status)
		row = append(row, instance.Os)
		row = append(row, instance.Region)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from v
func (v *Vultr) DeleteInstance(ctx *lepton.Context, instanceID string) error {
	err := v.Client.Instance.Delete(context.TODO(), instanceID)
	if err != nil {
		return err
	}
	return nil
}

// StartInstance starts an instance in v
func (v *Vultr) StartInstance(ctx *lepton.Context, instanceID string) error {

	err := v.Client.Instance.Start(context.TODO(), instanceID)
	if err != nil {
		return err
	}

	return nil
}

// StopInstance halts instance from v
func (v *Vultr) StopInstance(ctx *lepton.Context, instanceID string) error {
	err := v.Client.Instance.Halt(context.TODO(), instanceID)
	if err != nil {
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *Vultr) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := v.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Vultr) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}
