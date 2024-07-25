//go:build vultr || !onlyprovider

package vultr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
	"github.com/vultr/govultr/v3"
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

	zone := stripZone(c.CloudConfig.Zone)

	ig := &govultr.InstanceCreateReq{
		Region:     zone,
		Plan:       flavor,
		SnapshotID: c.CloudConfig.ImageName,
		Tags:       []string{"created-by-ops"},
	}

	cloudConfig := ctx.Config().CloudConfig
	if cloudConfig.StaticIP != "" {
		ips, _, _, err := v.Client.ReservedIP.List(context.TODO(), nil)
		if err != nil {
			fmt.Println(err)
		}

		ipUUID := ""
		for i := 0; i < len(ips); i++ {
			if ips[i].Subnet == cloudConfig.StaticIP {
				ipUUID = ips[i].ID
			}
		}

		if ipUUID != "" {
			ig.ReservedIPv4 = ipUUID
		} else {
			return fmt.Errorf("can't find the specified reserved ip")
		}
	}

	instance, res, err := v.Client.Instance.Create(context.TODO(), ig)
	if err != nil {
		fmt.Println(res)
		return err
	}

	if c.CloudConfig.DomainName != "" {
		ip := ""

		for i := 0; i < 15; i++ {
			server, _, err := v.Client.Instance.Get(context.TODO(), instance.ID)
			if err != nil {
				fmt.Println(err)
			}

			if server.MainIP != "0.0.0.0" {
				ip = server.MainIP
				break
			}

			time.Sleep(time.Millisecond * 500)
		}

		dn := c.CloudConfig.DomainName

		options := &govultr.ListOptions{}
		records, _, _, err := v.Client.DomainRecord.List(context.TODO(), dn, options)
		if err != nil {
			fmt.Println(err)
		}

		arec := ""
		for i := 0; i < len(records); i++ {
			if records[i].Type == "A" {
				arec = records[i].ID
			}
		}

		p := 0
		r := &govultr.DomainRecordReq{
			Name:     dn,
			Type:     "A",
			Data:     ip,
			TTL:      300,
			Priority: &p,
		}
		err = v.Client.DomainRecord.Update(context.TODO(), dn, arec, r)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (v *Vultr) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	instance, _, err := v.Client.Instance.Get(context.TODO(), name)
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
	instances, _, _, err := v.Client.Instance.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
		Tag:     "created-by-ops",
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

	instances, _, _, err := v.Client.Instance.List(context.TODO(), &govultr.ListOptions{
		PerPage: 100,
		Cursor:  "",
		Tag:     "created-by-ops",
	})
	if err != nil {
		return err
	}

	log.Debug("instances:", instances)
	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(instances)
	}

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

// RebootInstance reboots the instance.
func (v *Vultr) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
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

// InstanceStats show metrics for instances on vultr.
func (v *Vultr) InstanceStats(ctx *lepton.Context) error {
	return errors.New("currently not avilable")
}
