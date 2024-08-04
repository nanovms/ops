//go:build linode || !onlyprovider

package linode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/linode/linodego"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

func (v *Linode) addDisk(instanceID int, imageID int, imgName string) (int, error) {
	diskOpts := linodego.InstanceDiskCreateOptions{
		Label:    imgName,
		Image:    "private/" + strconv.Itoa(imageID),
		Size:     1300,
		RootPass: "__aC0mpl3xP@ssW0rd123__",
	}

	disk, err := v.Client.CreateInstanceDisk(context.Background(), instanceID, diskOpts)
	if err != nil {
		return 0, err
	}

	for i := 0; i < 30; i++ {
		status := v.getStatusForDisk(instanceID, disk.ID)
		time.Sleep(2 * time.Second) // hack

		if status == linodego.DiskReady {
			return disk.ID, nil
		}
	}
	return 0, fmt.Errorf("err: timed out waiting for disk to be ready")
}

func (v *Linode) addConfig(instanceID int, diskID int, imgName string) {
	createOpts := linodego.InstanceConfigCreateOptions{
		Devices: linodego.InstanceConfigDeviceMap{
			SDA: &linodego.InstanceConfigDevice{DiskID: diskID},
		},
		Kernel:   "linode/direct-disk",
		Label:    imgName,
		RunLevel: "default",
		VirtMode: "paravirt",
		Comments: "example config comment",
		// RootDevice: "/dev/sda",
		Helpers: &linodego.InstanceConfigHelpers{
			UpdateDBDisabled:  false,
			Distro:            false,
			ModulesDep:        false,
			Network:           true,
			DevTmpFsAutomount: false,
		},
	}
	config, err := v.Client.CreateInstanceConfig(context.Background(), instanceID, createOpts)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = v.Client.UpdateInstanceConfig(context.Background(), instanceID, config.ID, linodego.InstanceConfigUpdateOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (v *Linode) fetchImageID(ctx *lepton.Context) (int, error) {
	imgs, err := v.GetImages(ctx, "")
	if err != nil || len(imgs) == 0 {
		return 0, fmt.Errorf("error fetching image id: %w", err)
	}

	imgID := 0

	for i := 0; i < len(imgs); i++ {
		if imgs[i].Name == ctx.Config().CloudConfig.ImageName {
			s := strings.Split(imgs[i].ID, "private/")
			imgID, err = strconv.Atoi(s[1])
			if err != nil {
				return 0, err
			}
		}
	}

	return imgID, nil
}

// CreateInstance - Creates instance on Linode Platform
func (v *Linode) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()
	t := time.Now().Unix()
	st := strconv.FormatInt(t, 10)
	imgName := c.CloudConfig.ImageName + "-" + st
	fmt.Printf("creating instance %s\n", imgName)

	booted := false
	swapSize := 512
	flavor := "g6-nanode-1"
	zone := "us-west"
	if c.CloudConfig.Flavor != "" {
		flavor = c.CloudConfig.Flavor
	}
	if c.CloudConfig.Zone != "" {
		zone = c.CloudConfig.Zone
	}

	instance := linodego.InstanceCreateOptions{
		Label:    imgName,
		Region:   zone,
		Type:     flavor,
		RootPass: "__aC0mpl3xP@ssW0rd123__",
		Booted:   &booted,
		SwapSize: &swapSize,
	}
	linode, err := v.Client.CreateInstance(context.Background(), instance)
	if err != nil {
		return fmt.Errorf("error creating instance: %w", err)
	}

	for i := 0; i < 30; i++ {
		status := v.getStatusForLinode(linode.ID)
		time.Sleep(2 * time.Second) // hack

		if status == linodego.InstanceOffline {
			break
		}
	}

	imgID, err := v.fetchImageID(ctx)

	diskID, err := v.addDisk(linode.ID, imgID, imgName)
	if err != nil {
		return fmt.Errorf("error adding disk: %w", err)
	}

	v.addConfig(linode.ID, diskID, imgName)
	sinstanceID := strconv.Itoa(linode.ID)
	v.StartInstance(ctx, sinstanceID)

	return nil
}

func (v *Linode) getStatusForDisk(instanceID, diskID int) linodego.DiskStatus {
	disk, err := v.Client.GetInstanceDisk(context.Background(), instanceID, diskID)
	if err != nil {
		fmt.Println(err)
	}

	return disk.Status
}

func (v *Linode) getStatusForLinode(id int) linodego.InstanceStatus {
	instance, err := v.Client.GetInstance(context.Background(), id)
	if err != nil {
		fmt.Println(err)
	}

	return instance.Status
}

// GetInstanceByName returns instance with given name
func (v *Linode) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstances return all instances on Linode
func (v *Linode) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	instances, err := v.Client.ListInstances(context.Background(), &linodego.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var cloudInstances []lepton.CloudInstance

	for _, instance := range instances {
		cloudInstances = append(cloudInstances, lepton.CloudInstance{
			ID:        strconv.Itoa(instance.ID),
			Status:    string(instance.Status),
			Created:   instance.Created.Format(time.RFC3339),
			PublicIps: []string{instance.IPv4[0].String()},
			Image:     instance.Image, // made from disk not image so prob need to get from disk which is not actually in this call..
		})
	}

	return cloudInstances, nil
}

// ListInstances lists instances on v
func (v *Linode) ListInstances(ctx *lepton.Context) error {
	instances, err := v.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "MainIP", "Status", "ImageID"}) // "Region"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range instances {
		var row []string
		row = append(row, instance.ID)
		row = append(row, instance.PublicIps[0])
		row = append(row, instance.Status)
		row = append(row, instance.Image) /// Os)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from linode
func (v *Linode) DeleteInstance(ctx *lepton.Context, instanceID string) error {
	linodeID, err := strconv.Atoi(instanceID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	v.Client.DeleteInstance(context.Background(), linodeID)

	return nil
}

// RebootInstance reboots the instance.
func (v *Linode) RebootInstance(ctx *lepton.Context, instanceName string) error {
	filter := fmt.Sprintf("{\"label\": \"%s\"}", instanceName)
	opts := linodego.NewListOptions(0, filter)
	linodes, err := v.Client.ListInstances(context.Background(), opts)
	if err != nil || len(linodes) == 0 {
		return fmt.Errorf("error fetching linode id by name")
	}

	err = v.Client.RebootInstance(context.Background(), linodes[0].ID, 0)
	if err != nil {
		return fmt.Errorf("error rebooting instance: %w", err)
	}

	return nil
}

// StartInstance starts an instance in linode
func (v *Linode) StartInstance(ctx *lepton.Context, instanceID string) error {
	linodeID, err := strconv.Atoi(instanceID)
	if err != nil {
		return fmt.Errorf("error converting instanceID to int: %w", err)
	}

	err = v.Client.BootInstance(context.Background(), linodeID, 0)
	if err != nil {
		return fmt.Errorf("error booting instance: %w", err)
	}

	return nil
}

// StopInstance halts instance from v
func (v *Linode) StopInstance(ctx *lepton.Context, instanceID string) error {
	linodeID, err := strconv.Atoi(instanceID)
	if err != nil {
		return fmt.Errorf("error converting instanceID to str: %w", err)
	}

	err = v.Client.ShutdownInstance(context.Background(), linodeID)
	if err != nil {
		return fmt.Errorf("error shutting down instance: %w", err)
	}
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *Linode) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Linode) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// InstanceStats show metrics for instances on Linnode
func (v *Linode) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not avilable")
}
