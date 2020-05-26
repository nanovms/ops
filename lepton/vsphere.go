package lepton

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/vmware/govmomi/vim25/methods"

	"github.com/vmware/govmomi/vim25/types"

	"github.com/vmware/govmomi/find"

	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vmdk"
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

	vmdkPath := "/tmp/" + ctx.config.CloudConfig.ImageName + ".vmdk"

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), "/ha-datacenter/datastore/nanos-test")
	if err != nil {
		fmt.Println(err)
		return err
	}

	p := vmdk.ImportParams{
		Path: vmdkPath,
		//Logger:     logger,
		Type: "", // TODO: flag
		//Force: cmd.force,
		//Datacenter: dc,
		//Pool:       pool,
		//Folder:     folder,
	}

	// "/ha-datacenter/datastore/nanos-test"

	/*
		var ds = &object.Datastore{
			DatacenterPath: "/ha-datacenter",
		}
	*/

	err = vmdk.Import(context.TODO(), v.client, vmdkPath, ds, p)
	if err != nil {
		fmt.Println(err)
	}

	/*
		if err := v.client.ImportVmdk(vmdkPath, vsphereVolumeDir); err != nil {
			return nil, errors.New("importing data.vmdk to vsphere datastore", err)
		}
	*/

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

// CreateInstance - Creates instance on VSphere
func (v *Vsphere) CreateInstance(ctx *Context) error {
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
