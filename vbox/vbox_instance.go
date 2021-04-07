package vbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/wsl"
	"github.com/olekukonko/tablewriter"
	vbox "github.com/terra-farm/go-virtualbox"
)

var (
	virtualBoxDir = path.Join(lepton.GetOpsHome(), "virtualbox")
	vmsDir        = path.Join(lepton.GetOpsHome(), "virtualbox", "vms")
)

// CreateInstance uses an image to launch a vm in VirtualBox
func (p *Provider) CreateInstance(ctx *lepton.Context) (err error) {
	name := ctx.Config().RunConfig.InstanceName

	vmPath, err := findOrCreateVmsDir()
	if err != nil {
		return
	}

	imagesDir, err := findOrCreateVdiImagesDir()
	if err != nil {
		return
	}

	imageName := ctx.Config().CloudConfig.ImageName + ".vdi"
	imagePath := path.Join(imagesDir, imageName)
	vmImagePath := path.Join(vmPath, imageName)

	if wsl.IsWSL() {
		vmPath, err = wsl.ConvertPathFromWSLtoWindows(vmPath)
		if err != nil {
			return
		}

		imagePath, err = wsl.ConvertPathFromWSLtoWindows(imagePath)
		if err != nil {
			return
		}

		vmImagePath = vmPath + "\\" + imageName
	}

	err = vbox.CloneHD(imagePath, vmImagePath)
	if err != nil {
		return
	}

	vm, err := vbox.CreateMachine(name, vmPath)
	if err != nil {
		return
	}

	err = vm.AddStorageCtl("IDE", vbox.StorageController{
		SysBus:      vbox.SysBusIDE,
		Ports:       2,
		Chipset:     vbox.CtrlPIIX3,
		HostIOCache: true,
		Bootable:    true,
	})
	if err != nil {
		return
	}

	err = vm.AttachStorage("IDE", vbox.StorageMedium{
		Port:      uint(0),
		Device:    0,
		DriveType: vbox.DriveHDD,
		Medium:    vmImagePath,
	})
	if err != nil {
		return
	}

	modifyVm := exec.Command("VBoxManage", "modifyvm", vm.Name, "--memory", "1024")

	err = modifyVm.Run()
	if err != nil {
		return
	}

	err = vm.SetNIC(1, vbox.NIC{
		Network:  vbox.NICNetNAT,
		Hardware: vbox.VirtIO,
	})
	if err != nil {
		return
	}

	err = vm.Start()
	if err != nil {
		return
	}

	for _, p := range ctx.Config().RunConfig.Ports {
		port, err := strconv.Atoi(p)
		if err == nil {
			err = vm.AddNATPF(1, fmt.Sprintf("TCP:%d", port), vbox.PFRule{
				Proto:     vbox.PFTCP,
				HostPort:  uint16(port),
				GuestPort: uint16(port),
			})
			if err != nil {
				return err
			}
		}
	}

	return
}

// ListInstances prints vms list managed by VirtualBox in table
func (p *Provider) ListInstances(ctx *lepton.Context) (err error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Status", "Private Ips", "Public Ips", "Image"})
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

		rows = append(rows, i.ID)
		rows = append(rows, i.Name)
		rows = append(rows, i.Status)
		rows = append(rows, strings.Join(i.PrivateIps, ", "))
		rows = append(rows, strings.Join(i.PublicIps, ", "))
		rows = append(rows, i.Image)

		table.Append(rows)
	}

	table.Render()

	return
}

// GetInstances returns the list of vms managed by VirtualBox
func (p *Provider) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	instances = []lepton.CloudInstance{}

	vms, err := vbox.ListMachines()
	if err != nil {
		return
	}

	for _, vm := range vms {
		instances = append(instances, lepton.CloudInstance{
			ID:     vm.UUID,
			Name:   vm.Name,
			Status: string(vm.State),
		})
	}

	return
}

// DeleteInstance removes a vm
func (p *Provider) DeleteInstance(ctx *lepton.Context, instancename string) (err error) {

	vm, err := vbox.GetMachine(instancename)
	if err != nil {
		return err
	}

	err = vm.Delete()

	return
}

// StopInstance stops vm in VirtualBox
func (p *Provider) StopInstance(ctx *lepton.Context, instancename string) (err error) {
	vm, err := vbox.GetMachine(instancename)
	if err != nil {
		return err
	}

	modifyVm := exec.Command("VBoxManage", "controlvm", vm.Name, "poweroff")

	err = modifyVm.Run()

	return
}

// StartInstance starts a vm in VirtualBox
func (p *Provider) StartInstance(ctx *lepton.Context, instancename string) (err error) {
	vm, err := vbox.GetMachine(instancename)
	if err != nil {
		return err
	}

	err = vm.Start()

	return
}

// GetInstanceByID return a VirtualBox vm details with ID specified
func (p *Provider) GetInstanceByID(ctx *lepton.Context, id string) (instance *lepton.CloudInstance, err error) {
	return
}

// GetInstanceLogs is a stub
func (p *Provider) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", errors.New("Unsupported")
}

// PrintInstanceLogs prints vm log on console
func (p *Provider) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("Unsupported")
}

func findOrCreateVmsDir() (string, error) {
	if _, err := os.Stat(virtualBoxDir); os.IsNotExist(err) {
		os.MkdirAll(virtualBoxDir, 0755)
	} else if err != nil {
		return "", err
	}

	if _, err := os.Stat(vmsDir); os.IsNotExist(err) {
		os.MkdirAll(vmsDir, 0755)
	} else if err != nil {
		return "", err
	}

	return vmsDir, nil
}
