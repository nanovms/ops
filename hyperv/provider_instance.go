package hyperv

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance uses a vhdx image to launch a virtual machine in hyper-v
func (p *Provider) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	vmName := c.RunConfig.InstanceName
	imagePath := path.Join(vhdxImagesDir, c.CloudConfig.ImageName+".vhdx")

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf(`image with name "%s" not found`, c.CloudConfig.ImageName)
	}

	windowsImagePath, err := convertPathFromWSLtoWindows(imagePath)
	if err != nil {
		return err
	}

	err = CreateVirtualMachineWithNoHD(vmName, int64(1024*math.Pow10(6)), 1)
	if err != nil {
		return err
	}

	vmSwitch, err := GetExternalOnlineVirtualSwitch()
	if err != nil {
		return err
	}

	if vmSwitch == "" {
		err = CreateExternalVirtualSwitch("External")
		if err != nil {
			return err
		}
		vmSwitch = "External"
	}

	err = AddVirtualMachineNetworkAdapter(vmName, vmSwitch)
	if err != nil {
		return err
	}

	err = AddVirtualMachineHardDiskDrive(vmName, windowsImagePath)
	if err != nil {
		return err
	}

	err = SetVMComPort(vmName)
	if err != nil {
		return err
	}

	return StartVirtualMachine(vmName)
}

// ListInstances prints virtual machines list managed by hyper-v in table
func (p *Provider) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Port"})
	table.SetHeaderColor(
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
		rows = append(rows, i.Created)
		rows = append(rows, strings.Join(i.PrivateIps, ","))
		rows = append(rows, strings.Join(i.PublicIps, ","))

		table.Append(rows)
	}

	table.Render()

	return nil
}

// GetInstances returns the list of virtual machines managed by hyper-v
func (p *Provider) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	output, err := GetVirtualMachines()
	if err != nil {
		return
	}

	resp := []string{}

	s := bufio.NewScanner(strings.NewReader(output))
	for s.Scan() {
		resp = append(resp, s.Text())
	}

	if len(resp) > 2 {

		for i := 2; i < len(resp); i++ {
			line := resp[i]

			// remove duplicated spaces if they exist
			space := regexp.MustCompile(`\s+`)
			lineTrimmed := space.ReplaceAllString(line, " ")

			cols := strings.Split(lineTrimmed, " ")

			date, err := time.Parse("02/01/2006 15:04:05", cols[2]+" "+cols[3])
			if err != nil {
				fmt.Println(err)
				continue
			}

			instances = append(instances, lepton.CloudInstance{
				Name:    cols[0],
				Status:  cols[1],
				Created: lepton.Time2Human(date),
			})
		}

	}

	return
}

// DeleteInstance removes virtual machine
func (p *Provider) DeleteInstance(ctx *lepton.Context, instancename string) error {
	return DeleteVirtualMachine(instancename)
}

// StopInstance stops virtual machine in hyper-v
func (p *Provider) StopInstance(ctx *lepton.Context, instancename string) error {
	return StopVirtualMachine(instancename)
}

// StartInstance initiates virtual machine in hyper-v
func (p *Provider) StartInstance(ctx *lepton.Context, instancename string) error {
	return StartVirtualMachine(instancename)
}

// GetInstanceByID return a hyper-v virtual machine details with ID specified
func (p *Provider) GetInstanceByID(ctx *lepton.Context, id string) (*lepton.CloudInstance, error) {
	output, err := GetVirtualMachineByName(id)
	if err != nil {
		return nil, err
	}

	resp := []string{}

	s := bufio.NewScanner(strings.NewReader(output))
	for s.Scan() {
		resp = append(resp, s.Text())
	}

	if len(resp) > 2 {

		line := resp[2]

		// remove duplicated spaces if they exist
		space := regexp.MustCompile(`\s+`)
		lineTrimmed := space.ReplaceAllString(line, " ")

		cols := strings.Split(lineTrimmed, " ")

		date, err := time.Parse("02/01/2006 15:04:05", cols[2]+" "+cols[3])
		if err != nil {
			return nil, err
		}

		return &lepton.CloudInstance{
			Name:    cols[0],
			Status:  cols[1],
			Created: lepton.Time2Human(date),
		}, nil
	}
	return nil, fmt.Errorf(`vm with name "%s" not found`, id)
}

// GetInstanceLogs reads content from named pipe file
func (p *Provider) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", errors.New("Unsupported")
}

// PrintInstanceLogs prints vm logs content on console
func (p *Provider) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("Unsupported")
}
