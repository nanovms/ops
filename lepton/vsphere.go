package lepton

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/vmware/govmomi/find"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"

	"github.com/vmware/govmomi/vim25/types"

	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
)

// Vsphere provides access to the Vsphere API.
type Vsphere struct {
	Storage *Datastores
	client  *vim25.Client
}

// ResizeImage is not supported on VSphere.
func (v *Vsphere) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// BuildImage to be upload on VSphere
func (v *Vsphere) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.customizeImage(ctx)
}

// BuildImageWithPackage to upload on Vsphere.
func (v *Vsphere) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.customizeImage(ctx)
}

func (v *Vsphere) createImage(key string, bucket string, region string) {
	fmt.Println("private create")
}

// Initialize Vsphere related things
func (v *Vsphere) Initialize() error {
	venv := os.Getenv("GOVC_URL")

	u, err := url.Parse("https://" + venv + "/sdk")
	if err != nil {
		fmt.Println(err)
	}

	soapClient := soap.NewClient(u, true)
	v.client, err = vim25.NewClient(context.Background(), soapClient)
	if err != nil {
		fmt.Println(err)
	}

	req := types.Login{
		This: *v.client.ServiceContent.SessionManager,
	}

	req.UserName = u.User.Username()
	if pw, ok := u.User.Password(); ok {
		req.Password = pw
	}

	_, err = methods.Login(context.Background(), v.client, &req)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// CreateImage - Creates image on vsphere using nanos images
func (v *Vsphere) CreateImage(ctx *Context) error {

	// create new image w/ paravirtual && vmnetx3
	// this step prob belongs on creatinstance though..

	vmdkBase := strings.ReplaceAll(ctx.config.CloudConfig.ImageName, "-image", "")

	flatPath := "/tmp/" + vmdkBase + "-flat" + ".vmdk"
	imgPath := "/tmp/" + vmdkBase + ".vmdk"

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), "/ha-datacenter/datastore/datastore1/")
	if err != nil {
		fmt.Println(err)
		return err
	}

	p := soap.DefaultUpload
	ds.UploadFile(context.TODO(), flatPath, vmdkBase+"/"+vmdkBase+"-flat.vmdk", &p)
	ds.UploadFile(context.TODO(), imgPath, vmdkBase+"/"+vmdkBase+".vmdk", &p)

	dc, err := f.DatacenterOrDefault(context.TODO(), "/ha-datacenter/")
	if err != nil {
		fmt.Println(err)
		return err
	}

	m := ds.NewFileManager(dc, true)

	m.Copy(context.TODO(), vmdkBase+"/"+vmdkBase+".vmdk", vmdkBase+"/"+vmdkBase+"2.vmdk")

	return nil
}

// ListImages lists images on Vsphere
// ehhh.. these are really stopped vms
func (v *Vsphere) ListImages(ctx *Context) error {

	// Create view of VirtualMachine objects
	m := view.NewManager(v.client)

	cv, err := m.CreateContainerView(context.TODO(), v.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return err
	}

	defer cv.Destroy(context.TODO())

	// Retrieve summary property for all machines
	// Reference:
	// http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	var vms []mo.VirtualMachine
	err = cv.Retrieve(context.TODO(), []string{"VirtualMachine"}, []string{"summary"}, &vms)
	if err != nil {
		return err
	}

	// Print summary per vm (see also: govc/vm/info.go)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, vm := range vms {
		var row []string
		row = append(row, vm.Summary.Config.Name)
		row = append(row, "")
		row = append(row, "")
		row = append(row, "")
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from VSphere
func (v *Vsphere) DeleteImage(ctx *Context, imagename string) error {
	return nil
}

// CreateInstance - Creates instance on VSphere.
// Currently we support pvsci adapter && vmnetx3 network driver.
func (v *Vsphere) CreateInstance(ctx *Context) error {

	var devices object.VirtualDeviceList
	var err error

	controller := "pvscsi"

	spec := &types.VirtualMachineConfigSpec{
		Name:       "gtest", // FIXME: change hardcode
		GuestId:    "otherGuest64",
		NumCPUs:    1,
		MemoryMB:   1024,
		Annotation: "",
		Firmware:   string(types.GuestOsDescriptorFirmwareTypeBios),
		Version:    "",
	}

	// add disk
	scsi, err := devices.CreateSCSIController(controller)
	if err != nil {
		fmt.Println(err)
	}

	devices = append(devices, scsi)
	controller = devices.Name(scsi)

	dcontroller, err := devices.FindDiskController(controller)
	if err != nil {
		fmt.Println(err)
	}

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), "/ha-datacenter/datastore/datastore1/")
	if err != nil {
		fmt.Println(err)
		return err
	}

	// FIXME - change hard-code
	path := ds.Path("gtest/gtest2.vmdk")
	disk := devices.CreateDisk(dcontroller, ds.Reference(), path)

	disk = devices.ChildDisk(disk)

	devices = append(devices, disk)

	// end add disk

	// add network
	// infer network stub
	net, err := f.NetworkOrDefault(context.TODO(), "/ha-datacenter/network/VM Network")
	if err != nil {
		fmt.Println(err)
	}

	backing, err := net.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		fmt.Println(err)
	}

	device, err := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", backing)
	if err != nil {
		fmt.Println(err)
	}

	netdev := device

	devices = append(devices, netdev)

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		fmt.Println(err)
	}

	spec.DeviceChange = deviceChange

	var datastore *object.Datastore

	datastore = ds

	dc, err := f.DatacenterOrDefault(context.TODO(), "/ha-datacenter/")
	if err != nil {
		fmt.Println(err)
		return err
	}

	folders, err := dc.Folders(context.TODO())
	if err != nil {
		fmt.Println(err)
	}

	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastore.Name()),
	}

	folder := folders.VmFolder

	pool, err := f.ResourcePoolOrDefault(context.TODO(), "/ha-datacenter/host/localhost.hsd1.ca.comcast.net/Resources")
	if err != nil {
		fmt.Println(err)
	}

	task, err := folder.CreateVM(context.TODO(), *spec, pool, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	info, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		fmt.Printf("%+v", info)
		fmt.Printf("%+v", info.Reason)
		fmt.Println(err)
		return err
	}

	object.NewVirtualMachine(v.client, info.Result.(types.ManagedObjectReference))

	return nil
}

// ListInstances lists instances on VSphere
func (v *Vsphere) ListInstances(ctx *Context) error {

	// Create a view of HostSystem objects
	m := view.NewManager(v.client)

	v2, err := m.CreateContainerView(context.TODO(), v.client.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		fmt.Println(err)
		return err
	}

	defer v2.Destroy(context.TODO())

	// Retrieve summary property for all hosts
	// Reference:
	// http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.HostSystem.html
	var hss []mo.HostSystem
	err = v2.Retrieve(context.TODO(), []string{"HostSystem"}, []string{"summary"}, &hss)
	if err != nil {
		return err
	}

	// Print summary per host (see also: govc/host/info.go)

	fmt.Println("woot?")
	fmt.Printf("%+v", hss)
	/*
		tw := tablewriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "Name:\tUsed CPU:\tTotal CPU:\tFree CPU:\tUsed Memory:\tTotal Memory:\tFree Memory:\t\n")

		for _, hs := range hss {
			totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
			freeCPU := int64(totalCPU) - int64(hs.Summary.QuickStats.OverallCpuUsage)
			freeMemory := int64(hs.Summary.Hardware.MemorySize) - (int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024)
			fmt.Fprintf(tw, "%s\t", hs.Summary.Config.Name)
			fmt.Fprintf(tw, "%d\t", hs.Summary.QuickStats.OverallCpuUsage)
			fmt.Fprintf(tw, "%d\t", totalCPU)
			fmt.Fprintf(tw, "%d\t", freeCPU)
			fmt.Fprintf(tw, "%s\t", hs.Summary.QuickStats.OverallMemoryUsage)
			fmt.Fprintf(tw, "%s\t", hs.Summary.Hardware.MemorySize)
			fmt.Fprintf(tw, "%d\t", freeMemory)
			fmt.Fprintf(tw, "\n")
		}

		_ = tw.Flush()
	*/
	return nil
}

// DeleteInstance deletes instance from VSphere
func (v *Vsphere) DeleteInstance(ctx *Context, instancename string) error {
	return nil
}

// StartInstance starts an instance in VSphere
func (v *Vsphere) StartInstance(ctx *Context, instancename string) error {
	// govc vm.power -on=true gtest

	return nil
}

// StopInstance deletes instance from VSphere
func (v *Vsphere) StopInstance(ctx *Context, instancename string) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Vsphere) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {
	return nil
}

// Todo - make me shared
func (v *Vsphere) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}
