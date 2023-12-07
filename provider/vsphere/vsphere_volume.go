//go:build vsphere || !onlyprovider

package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vslm"
)

// CreateVolume converts local volume raw file to mkfs and uploads required files to datastore
func (v *Vsphere) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	config := ctx.Config()
	var vol lepton.NanosVolume

	vol, err := lepton.CreateLocalVolume(config, name, data, provider)
	if err != nil {
		return vol, err
	}

	bucket := "volumes"
	config.CloudConfig.ImageName = vol.Name

	err = v.Storage.CopyToBucket(config, vol.Path)
	if err != nil {
		return vol, err
	}

	vmdkPath := "/tmp/" + vol.Name + ".vmdk"
	flatVmdkPath := "/tmp/" + vol.Name + "-flat.vmdk"

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		fmt.Println(err)
		return vol, err
	}

	p := soap.DefaultUpload
	err = ds.UploadFile(context.TODO(), vmdkPath, bucket+"/"+vol.Name+".vmdk", &p)
	if err != nil {
		return vol, err
	}

	err = ds.UploadFile(context.TODO(), flatVmdkPath, bucket+"/"+vol.Name+"-flat.vmdk", &p)
	if err != nil {
		return vol, err
	}

	objectManager := vslm.NewObjectManager(ds.Client())

	_, err = objectManager.RegisterDisk(context.TODO(), ds.NewURL("volumes/"+vol.Name+".vmdk").String(), vol.Name)
	if err != nil {
		return vol, fmt.Errorf("register disk: %v", err)
	}

	return vol, nil
}

// getAllVolumes uses object manager to get volumes registered and return them
func (v *Vsphere) getAllVolumes(ds *object.Datastore) (*[]types.VStorageObject, error) {
	disks := &[]types.VStorageObject{}

	objectManager := vslm.NewObjectManager(ds.Client())

	ids, err := objectManager.List(context.TODO(), ds)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		obj, err := objectManager.Retrieve(context.TODO(), ds, id.Id)
		if err != nil && err.Error() == "ServerFaultCode: The object or item referred to could not be found." {
			fmt.Printf("object with id %s not found: %s\n", id.Id, err.Error())
		} else if err != nil {
			return nil, err
		} else {
			*disks = append(*disks, *obj)
		}
	}

	return disks, nil
}

// GetAllVolumes returns volumes. Work in progress
func (v *Vsphere) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	vols := &[]lepton.NanosVolume{}

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		return nil, err
	}

	// List files inside volumes directory in datastore
	// TODO: Get all virtual machines and check if volumes are attached to them
	browser, err := ds.Browser(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("datastore browser: %v", err)
	}

	spec := &types.HostDatastoreBrowserSearchSpec{}

	task, err := browser.SearchDatastore(context.TODO(), ds.Path("volumes"), spec)
	if err != nil {
		return nil, fmt.Errorf("datastore browser search: %v", err)
	}

	taskInfo, err := task.WaitForResult(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("waiting for datastore search: %v", err)
	}

	files := taskInfo.Result.(types.HostDatastoreBrowserSearchResults).File

	for _, file := range files {
		path := file.GetFileInfo().Path
		if !strings.Contains(path, "-flat.vmdk") {
			*vols = append(*vols, lepton.NanosVolume{Name: path})
		}
	}

	// TODO: use object manager to get all volumes listed and convert them to nanos volumes, blocked by https://github.com/vmware/govmomi/issues/2174
	disks, err := v.getAllVolumes(ds)
	if err != nil {
		return nil, fmt.Errorf("get all volumes: %v", disks)
	}

	return vols, nil
}

// DeleteVolume deletes a volume on vsphere
func (v *Vsphere) DeleteVolume(ctx *lepton.Context, name string) (err error) {
	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		return err
	}

	objectManager := vslm.NewObjectManager(ds.Client())

	disks, err := v.getAllVolumes(ds)
	if err != nil {
		return fmt.Errorf("get all volumes: %v", disks)
	}

	for _, disk := range *disks {
		if disk.Config.Name == name {
			task, err := objectManager.Delete(context.TODO(), ds, disk.Config.Id.Id)
			if err != nil {
				return err
			}

			err = task.Wait(context.TODO())
			if err != nil {
				return fmt.Errorf("deleting %s: %v", disk.Config.Id.Id, err)
			}
		}
	}

	return
}

// AttachVolume attaches a volume to an instance
func (v *Vsphere) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		return err
	}

	vm, err := v.getVirtualMachine(image)
	if err != nil {
		return err
	}

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}

	controller, err := devices.FindDiskController("")
	if err != nil {
		return err
	}

	disk := devices.CreateDisk(controller, ds.Reference(), ds.Path("volumes/"+name))

	err = vm.AddDevice(context.TODO(), disk)
	if err != nil {
		return err
	}

	task, err := vm.Reset(context.TODO())
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

// DetachVolume detaches a volume from an instance
func (v *Vsphere) DetachVolume(ctx *lepton.Context, image, name string) error {
	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		return err
	}

	vm, err := v.getVirtualMachine(image)
	if err != nil {
		return err
	}

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}

	query := fmt.Sprintf("[%s] %s/%s.vmdk", ds.Name(), "volumes", name)

	var device types.BaseVirtualDevice
	for _, dev := range devices {
		backing := dev.GetVirtualDevice().Backing

		if backing != nil {
			info, ok := backing.(types.BaseVirtualDeviceFileBackingInfo)
			if ok && query == info.GetVirtualDeviceFileBackingInfo().FileName {
				device = dev
				break
			}
		}
	}

	if device == nil {
		return lepton.ErrVolumeNotFound(query)
	}

	err = vm.RemoveDevice(context.TODO(), true, device)
	if err != nil {
		return err
	}

	task, err := vm.Reset(context.TODO())
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO())
	if err != nil {
		return err
	}

	return nil
}

func (v *Vsphere) getVirtualMachine(instanceName string) (*object.VirtualMachine, error) {
	m := view.NewManager(v.client)

	cv, err := m.CreateContainerView(context.TODO(), v.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	defer cv.Destroy(context.TODO())

	var vms []mo.VirtualMachine
	err = cv.RetrieveWithFilter(context.TODO(), []string{"VirtualMachine"}, []string{"summary"}, &vms, property.Filter{"name": instanceName})
	if err != nil {
		return nil, err
	}

	if len(vms) == 0 {
		return nil, lepton.ErrInstanceNotFound(instanceName)
	}

	return object.NewVirtualMachine(v.client, vms[0].Reference()), nil
}

// CreateVolumeImage ...
func (v *Vsphere) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (v *Vsphere) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
