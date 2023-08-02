//go:build vultr || !onlyprovider

package vultr

import (
	"context"
	"encoding/json"
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
		/*
			 this,
			 https://github.com/vultr/govultr/blob/master/instance_test.go#L1156
			 is actually wrong:
			 1)  it takes a UUID of the ip, which we need
			 to slurp in from here:

			  curl "https://api.vultr.com/v2/reserved-ips" \\n  -X GET \\n
			  -H "Authorization: Bearer token" | jq

			2) then after you plug it here you must attach in separate call

			  curl
			  "https://api.vultr.com/v2/reserved-ips/some-uuid/attach"
			  \\n  -X POST \\n  -H "Authorization: Bearer token-goes-here" \\n  -H "Content-Type:
			  application/json" \\n  --data '{\n    "instance_id" :
			  "some-uuid"\n  }'

			 3) finally you have to reboot the instance
			 https://www.vultr.com/api/#operation/reboot-instance
		*/
		fmt.Printf("setting %s\n", cloudConfig.StaticIP)
		ig.ReservedIPv4 = cloudConfig.StaticIP
	}

	instance, res, err := v.Client.Instance.Create(context.TODO(), ig)
	if err != nil {
		fmt.Println(res)
		return err
	}

	log.Info("instance:", instance)

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
