//go:build vsphere || !onlyprovider

package vsphere

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// CreateInstance - Creates instance on VSphere.
// Currently we support pvsci adapter && vmnetx3 network driver.
func (v *Vsphere) CreateInstance(ctx *lepton.Context) error {

	var devices object.VirtualDeviceList
	var err error

	imgName := ctx.Config().CloudConfig.ImageName

	fmt.Printf("spinning up:\t%s\n", imgName)

	spec := &types.VirtualMachineConfigSpec{
		Name:       imgName,
		GuestId:    "otherGuest64",
		NumCPUs:    1,
		MemoryMB:   1024,
		Annotation: "",
		Firmware:   string(types.GuestOsDescriptorFirmwareTypeBios),
		Version:    "",
	}

	// add disk
	scsi, err := devices.CreateSCSIController("pvscsi")
	if err != nil {
		log.Error(err)
	}

	devices = append(devices, scsi)
	controller := devices.Name(scsi)

	dcontroller, err := devices.FindDiskController(controller)
	if err != nil {
		log.Error(err)
	}

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		log.Error(err)
		return err
	}

	dpath := ds.Path(imgName + "/" + imgName + "2.vmdk")
	_, err = ds.Stat(context.TODO(), dpath)
	if err != nil {
		log.Debug(err)
		return errors.New("Image " + imgName + " not found")
	}
	disk := devices.CreateDisk(dcontroller, ds.Reference(), dpath)

	disk = devices.ChildDisk(disk)

	devices = append(devices, disk)
	// end add disk

	// add network
	// infer network stub
	net, err := f.NetworkOrDefault(context.TODO(), v.network)
	if err != nil {
		log.Error(err)
	}

	backing, err := net.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		log.Error(err)
	}

	device, err := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", backing)
	if err != nil {
		log.Error(err)
	}

	devices = append(devices, device)

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		log.Error(err)
	}

	spec.DeviceChange = deviceChange

	var datastorez *object.Datastore

	datastorez = ds

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
		return err
	}

	folders, err := dc.Folders(context.TODO())
	if err != nil {
		log.Error(err)
	}

	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastorez.Name()),
	}

	folder := folders.VmFolder

	pool, err := f.ResourcePoolOrDefault(context.TODO(), v.resourcePool)
	if err != nil {
		log.Error(err)
		fmt.Println("Did you set the correct Resource Pool? https://nanovms.gitbook.io/ops/vsphere#create-instance ")
		os.Exit(1)
	}

	task, err := folder.CreateVM(context.TODO(), *spec, pool, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	info, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		fmt.Printf("%+v", info)
		fmt.Printf("%+v", info.Reason)
		log.Error(err)
		return err
	}

	vm := object.NewVirtualMachine(v.client, info.Result.(types.ManagedObjectReference))

	devices, err = vm.Device(context.TODO())
	if err != nil {
		return err
	}

	// add serial for logs
	serial, err := devices.CreateSerialPort()
	if err != nil {
		log.Error(err)
	}

	err = vm.AddDevice(context.TODO(), serial)
	if err != nil {
		return err
	}

	devices, err = vm.Device(context.TODO())
	if err != nil {
		return err
	}

	d, err := devices.FindSerialPort("")
	if err != nil {
		return err
	}

	devices = devices.SelectByType(d)

	var mvm mo.VirtualMachine
	err = vm.Properties(context.TODO(), vm.Reference(), []string{"config.files.logDirectory"}, &mvm)
	if err != nil {
		return err
	}

	uri := path.Join(mvm.Config.Files.LogDirectory, "console.log")

	err = vm.EditDevice(context.TODO(), devices.ConnectSerialPort(d, uri, false, ""))
	if err != nil {
		log.Error(err)
	}

	task, err = vm.PowerOn(context.TODO())
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (v *Vsphere) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	m := view.NewManager(v.client)

	cv, err := m.CreateContainerView(context.TODO(), v.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	defer cv.Destroy(context.TODO())

	var vms []mo.VirtualMachine
	err = cv.RetrieveWithFilter(context.TODO(), []string{"VirtualMachine"}, []string{"summary"}, &vms, property.Filter{"name": name})
	if err != nil {
		return nil, err
	}

	if len(vms) == 0 {
		return nil, lepton.ErrInstanceNotFound(name)
	}

	return v.convertToCloudInstance(&vms[0]), nil
}

// GetInstances return all instances on vSphere
func (v *Vsphere) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	var cinstances []lepton.CloudInstance

	m := view.NewManager(v.client)

	cv, err := m.CreateContainerView(context.TODO(), v.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	defer cv.Destroy(context.TODO())

	var vms []mo.VirtualMachine
	err = cv.Retrieve(context.TODO(), []string{"VirtualMachine"}, []string{"summary"}, &vms)
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		cInstance := v.convertToCloudInstance(&vm)

		cinstances = append(cinstances, *cInstance)
	}

	return cinstances, nil
}

func (v *Vsphere) convertToCloudInstance(vm *mo.VirtualMachine) *lepton.CloudInstance {
	cInstance := lepton.CloudInstance{
		Name:   vm.Summary.Config.Name,
		Status: string(vm.Summary.Runtime.PowerState),
	}

	if vm.Summary.Runtime.BootTime != nil {
		cInstance.Created = vm.Summary.Runtime.BootTime.String()
	}

	if cInstance.Status == "poweredOn" {
		ip := v.ipFor(vm.Summary.Config.Name)
		cInstance.PublicIps = []string{ip}
	}

	return &cInstance
}

// ListInstances lists instances on VSphere.
// It essentially does:
// govc ls /ha-datacenter/vm
func (v *Vsphere) ListInstances(ctx *lepton.Context) error {

	cInstances, err := v.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "IP", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range cInstances {
		var row []string

		row = append(row, instance.Name)

		row = append(row, strings.Join(instance.PublicIps, ","))

		row = append(row, instance.Status)
		row = append(row, instance.Created)
		table.Append(row)
	}

	table.Render()

	return nil
}

// govc vm.ip -esxcli -wait 5s dtest
// waits for up to 1hr!?? wtf
//
// if we get empty string set the following && try again
// govc host.esxcli system settings advanced set -o /Net/GuestIPHack -i 1
func (v *Vsphere) ipFor(instancename string) string {

	f := find.NewFinder(v.client, true)

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
	}

	f.SetDatacenter(dc)

	vm, err := f.VirtualMachine(context.TODO(), instancename)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			fmt.Println("can't find vm " + instancename)
		}
		log.Error(err)
	}

	var get func(*object.VirtualMachine) (string, error) = func(vm *object.VirtualMachine) (string, error) {

		guest := esxcli.NewGuestInfo(v.client)

		ticker := time.NewTicker(time.Millisecond * 500)
		defer ticker.Stop()

		icnt := 0

		for {
			select {
			case <-ticker.C:

				if icnt > 3 {
					v.setGuestIPHack()
				}

				ip, err := guest.IpAddress(vm)
				if err != nil {
					log.Error(err)
					return "", err
				}

				if ip != "0.0.0.0" {
					return ip, nil
				}

				icnt++

			}
		}
	}

	ip, err := get(vm)
	if err != nil {
		log.Error(err)
	}

	return ip
}

func (v *Vsphere) findHostPath() string {
	f := find.NewFinder(v.client, true)
	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
	}

	f.SetDatacenter(dc)

	host, err := f.DefaultHostSystem(context.TODO())
	if err != nil {
		log.Error(err)
	}

	return host.InventoryPath
}

func (v *Vsphere) runCLI(args []string) (*esxcli.Response, error) {
	f := find.NewFinder(v.client, true)

	hostPath := v.findHostPath()
	host, err := f.HostSystemOrDefault(context.TODO(), hostPath)
	if err != nil {
		log.Error(err)
	}

	e, err := esxcli.NewExecutor(v.client, host)
	if err != nil {
		log.Error(err)
	}

	return e.Run(args)
}

func (v *Vsphere) iphackEnabled() bool {
	args := []string{"system", "settings", "advanced", "list", "-o", "/Net/GuestIPHack"}
	res, err := v.runCLI(args)
	if err != nil {
		log.Error(err)
	}

	for _, val := range res.Values {
		if ival, ok := val["IntValue"]; ok {
			if ival[0] == "1" {
				return true
			}
		}
	}

	return false
}

func (v *Vsphere) setGuestIPHack() {
	if v.iphackEnabled() {
		log.Info("ip hack enabled")
	} else {
		log.Info("setting ip hack")

		args := []string{"system", "settings", "advanced", "set", "-o", "/Net/GuestIPHack", "-i", "1"}

		res, err := v.runCLI(args)
		if err != nil {
			log.Error(err)
		}

		debug := false // FIXME: should have a debug log throughout OPS
		if debug {
			for _, val := range res.Values {
				log.Debug(fmt.Sprint(val))
			}
		}
	}

	fmt.Println("IP hack has been enabled for all new ARP requests, however, for existing hosts the easiest way to trigger that is to simply reboot the vm.")
	os.Exit(0)
}

// DeleteInstance deletes instance from VSphere
func (v *Vsphere) DeleteInstance(ctx *lepton.Context, instancename string) error {
	f := find.NewFinder(v.client, true)

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
		return err
	}

	f.SetDatacenter(dc)

	vms, err := f.VirtualMachineList(context.TODO(), instancename)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			fmt.Println("can't find vm " + instancename)
		}
		log.Error(err)
	}

	vm := vms[0]

	task, err := vm.PowerOff(context.TODO())
	if err != nil {
		log.Error(err)
	}

	// Ignore error since the VM may already been in powered off
	// state.
	// vm.Destroy will fail if the VM is still powered on.
	_ = task.Wait(context.TODO())

	task, err = vm.Destroy(context.TODO())
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

// RebootInstance reboots the instance.
func (v *Vsphere) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance in VSphere.
// It is the equivalent of:
// govc vm.power -on=true <instance_name>
func (v *Vsphere) StartInstance(ctx *lepton.Context, instancename string) error {
	f := find.NewFinder(v.client, true)

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
		return err
	}

	f.SetDatacenter(dc)

	vms, err := f.VirtualMachineList(context.TODO(), instancename)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			fmt.Println("can't find vm " + instancename)
		}
		log.Error(err)
	}

	task, err := vms[0].PowerOn(context.TODO())
	if err != nil {
		log.Error(err)
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	return err
}

// StopInstance stops an instance from VSphere
// It is the equivalent of:
// govc vm.power -on=false <instance_name>
func (v *Vsphere) StopInstance(ctx *lepton.Context, instancename string) error {
	f := find.NewFinder(v.client, true)

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
		return err
	}

	f.SetDatacenter(dc)

	vms, err := f.VirtualMachineList(context.TODO(), instancename)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			fmt.Println("can't find vm " + instancename)
		}
		log.Error(err)
	}

	task, err := vms[0].PowerOff(context.TODO())
	if err != nil {
		log.Error(err)
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	return err
}

// PrintInstanceLogs writes instance logs to console
func (v *Vsphere) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := v.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs.
// govc datastore.tail -n 100 gtest/serial.out
// logs don't appear until you spin up the instance.
func (v *Vsphere) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		return "", err
	}

	serialFile := instancename + "/console.log"
	file, err := ds.Open(context.TODO(), serialFile)
	if err != nil {
		return "", err
	}
	var reader io.ReadCloser = file

	err = file.Tail(100)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, reader)

	_ = reader.Close()

	return buf.String(), nil
}

// InstanceStats show metrics for instances on vsphere.
func (p *Vsphere) InstanceStats(ctx *lepton.Context) error {
	return errors.New("currently not avilable")
}
