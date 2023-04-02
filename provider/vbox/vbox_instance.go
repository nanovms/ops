//go:build vbox || !onlyprovider

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
	"github.com/terra-farm/go-virtualbox"
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
		ctx.Logger().Errorf("failed converting vms path")
		return
	}

	vmPath = path.Join(vmPath, name)
	err = os.Mkdir(vmPath, 0755)
	if err != nil {
		ctx.Logger().Errorf("failed creating vm directory")
		return
	}

	imagesDir, err := findOrCreateVdiImagesDir()
	if err != nil {
		ctx.Logger().Errorf("failed converting vdi images dir")
		return
	}

	imageName := ctx.Config().CloudConfig.ImageName + ".vdi"
	imagePath := path.Join(imagesDir, imageName)
	vmImagePath := path.Join(vmPath, imageName)

	if wsl.IsWSL() {
		vmPath, err = wsl.ConvertPathFromWSLtoWindows(vmPath)
		if err != nil {
			ctx.Logger().Errorf("failed converting vm path")
			return
		}

		imagePath, err = wsl.ConvertPathFromWSLtoWindows(imagePath)
		if err != nil {
			ctx.Logger().Errorf("failed converting image path %s", imagePath)
			return
		}

		vmImagePath = vmPath + "\\" + imageName
	}

	err = vbox.CloneHD(imagePath, vmImagePath)
	if err != nil {
		ctx.Logger().Errorf("failed cloning hdd")
		return
	}

	vm, err := vbox.CreateMachine(name, vmPath)
	if err != nil {
		ctx.Logger().Errorf("failed creating machine")
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
		ctx.Logger().Errorf("failed adding storage controller")
		return
	}

	err = vm.AttachStorage("IDE", vbox.StorageMedium{
		Port:      uint(0),
		Device:    0,
		DriveType: vbox.DriveHDD,
		Medium:    vmImagePath,
	})
	if err != nil {
		ctx.Logger().Errorf("failed attaching storage")
		return
	}

	modifyVM := exec.Command("VBoxManage", "modifyvm", vm.Name, "--memory", "1024")

	err = modifyVM.Run()
	if err != nil {
		ctx.Logger().Errorf("failed changing vm memory")
		return
	}

	err = virtualbox.SetGuestProperty(vm.Name, "Image", imageName)
	if err != nil {
		ctx.Logger().Errorf("failed to set image as guest property")
		return
	}

	err = vm.SetNIC(1, vbox.NIC{
		Network:  vbox.NICNetNAT,
		Hardware: vbox.VirtIO,
	})
	if err != nil {
		ctx.Logger().Errorf("failed setting NIC")
		return
	}

	err = vm.Start()
	if err != nil {
		ctx.Logger().Errorf("failed started vm")
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
				ctx.Logger().Errorf("failed creating port forward rule")
				return err
			}
		}
	}

	// TODO: change go-virtualbox to read forwarding rules on listing instances instead of relying on a guest property
	if len(ctx.Config().RunConfig.Ports) != 0 {
		err = virtualbox.SetGuestProperty(vm.Name, "Ports", strings.Join(ctx.Config().RunConfig.Ports, ","))
		if err != nil {
			ctx.Logger().Errorf("failed to set ports as guest property")
			return
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
	table.SetHeader([]string{"ID", "Name", "Status", "Public Ips", "Ports", "Image"})
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
		rows = append(rows, strings.Join(i.PublicIps, ", "))
		rows = append(rows, strings.Join(i.Ports, ", "))
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
		image, _ := virtualbox.GetGuestProperty(vm.Name, "Image")
		ports, _ := virtualbox.GetGuestProperty(vm.Name, "Ports")

		instances = append(instances, lepton.CloudInstance{
			ID:        vm.UUID,
			Name:      vm.Name,
			PublicIps: []string{"127.0.0.1"},
			Status:    string(vm.State),
			Image:     image,
			Ports:     []string{ports},
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

	stopVMCmd := exec.Command("VBoxManage", "controlvm", vm.Name, "poweroff")

	err = stopVMCmd.Run()

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

// GetInstanceByName returns VirtualBox vm with given name
func (p *Provider) GetInstanceByName(ctx *lepton.Context, name string) (instance *lepton.CloudInstance, err error) {
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
