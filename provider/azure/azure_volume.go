//go:build azure || !onlyprovider

package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-07-02/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

// CreateVolume uploads the volume raw file and creates a disk from it
func (a *Azure) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	config := ctx.Config()

	var vol lepton.NanosVolume

	disksClient, err := a.getDisksClient()
	if err != nil {
		return vol, err
	}

	location := a.getLocation(config)

	var sizeInGb int64
	if config.BaseVolumeSz != "" {
		size, err := lepton.GetSizeInGb(config.BaseVolumeSz)
		if err != nil {
			return vol, fmt.Errorf("cannot get volume size: %v", err)
		}
		config.BaseVolumeSz = "" // create minimum-sized local volume
		sizeInGb = int64(size)
	}

	vol, err = lepton.CreateLocalVolume(config, name, data, provider)
	if err != nil {
		return vol, fmt.Errorf("create local volume: %v", err)
	}
	defer os.Remove(vol.Path)

	config.CloudConfig.ImageName = name

	err = a.Storage.CopyToBucket(config, vol.Path)
	if err != nil {
		return vol, fmt.Errorf("copy volume archive to azure bucket: %v", err)
	}

	bucket, err := a.getBucketName()
	if err != nil {
		return vol, err
	}

	container := "quickstart-nanos"
	disk := name + ".vhd"

	sourceURI := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	diskParams := compute.Disk{
		Location: to.StringPtr(location),
		Name:     to.StringPtr(name),
		DiskProperties: &compute.DiskProperties{
			HyperVGeneration: compute.V1,
			DiskSizeGB:       to.Int32Ptr(int32(sizeInGb)),
			CreationData: &compute.CreationData{
				CreateOption:     "Import",
				SourceURI:        to.StringPtr(sourceURI),
				StorageAccountID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s", a.subID, a.groupName, bucket)),
			},
		},
	}

	_, err = disksClient.CreateOrUpdate(context.TODO(), a.groupName, name, diskParams)
	if err != nil {
		return vol, err
	}

	return vol, nil
}

// GetAllVolumes returns all volumes in NanosVolume format
func (a *Azure) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	vols := &[]lepton.NanosVolume{}

	volumesService, err := a.getDisksClient()
	if err != nil {
		return nil, err
	}

	azureDisksPage, err := volumesService.List(context.TODO())
	if err != nil {
		return nil, err
	}

	for {
		disks := azureDisksPage.Values()

		if disks == nil {
			break
		}

		for _, disk := range disks {
			var attachedTo string
			if disk.ManagedBy != nil {
				instanceURLParts := strings.Split(*disk.ManagedBy, "/")

				attachedTo = instanceURLParts[len(instanceURLParts)-1]
			}

			vol := lepton.NanosVolume{
				Name:       *disk.Name,
				Status:     string(disk.DiskProperties.DiskState),
				Size:       strconv.Itoa(int(*disk.DiskSizeGB)),
				Path:       "",
				CreatedAt:  disk.TimeCreated.String(),
				AttachedTo: attachedTo,
			}

			*vols = append(*vols, vol)

			azureDisksPage.Next()
		}
	}

	return vols, nil
}

// DeleteVolume deletes an existing volume
func (a *Azure) DeleteVolume(ctx *lepton.Context, name string) error {
	volumesService, err := a.getDisksClient()
	if err != nil {
		return err
	}

	_, err = volumesService.Delete(context.TODO(), a.groupName, name)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches a volume to an instance
func (a *Azure) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	vmClient := a.getVMClient()

	vm, err := vmClient.Get(context.TODO(), a.groupName, image, compute.InstanceViewTypesInstanceView)
	if err != nil {
		return err
	}

	disksClient, err := a.getDisksClient()
	if err != nil {
		return err
	}

	disk, err := disksClient.Get(context.TODO(), a.groupName, name)
	if err != nil {
		return err
	}
	newDisk := compute.DataDisk{
		Lun:          to.Int32Ptr(0),
		Name:         &name,
		CreateOption: compute.DiskCreateOptionTypesAttach,
		ManagedDisk: &compute.ManagedDiskParameters{
			ID: to.StringPtr(*disk.ID),
		},
	}
	if attachID > 0 {
		if attachID < 64 {
			newDisk.Lun = to.Int32Ptr(int32(attachID))
		} else {
			return fmt.Errorf("invalid attachment point %d; allowed values: 0-63", attachID)
		}
	}
	*vm.StorageProfile.DataDisks = append(*vm.StorageProfile.DataDisks, newDisk)
	future, err := vmClient.CreateOrUpdate(context.TODO(), a.groupName, image, vm)
	if err != nil {
		return fmt.Errorf("cannot update vm: %v", err)
	}

	log.Info("attaching the volume - this can take a few minutes - you can ctrl-c this after a bit")

	err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	return nil
}

// DetachVolume detachs a volume from an instance
func (a *Azure) DetachVolume(ctx *lepton.Context, image, name string) error {
	vmClient := a.getVMClient()

	vm, err := vmClient.Get(context.TODO(), a.groupName, image, compute.InstanceViewTypesInstanceView)
	if err != nil {
		return err
	}

	dataDisks := &[]compute.DataDisk{}

	for _, disk := range *vm.StorageProfile.DataDisks {
		if *disk.Name != name {
			*dataDisks = append(*dataDisks, disk)
		}
	}

	vm.StorageProfile.DataDisks = dataDisks

	future, err := vmClient.CreateOrUpdate(context.TODO(), a.groupName, image, vm)
	if err != nil {
		return fmt.Errorf("cannot update vm: %v", err)
	}

	log.Info("detaching the volume - this can take a few minutes - you can ctrl-c this after a bit")

	err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the vm create or update future response: %v", err)
	}

	return nil
}

func (a *Azure) getDisksClient() (*compute.DisksClient, error) {
	vmClient := compute.NewDisksClientWithBaseURI(compute.DefaultBaseURI, a.subID)
	authr, err := a.GetResourceManagementAuthorizer()
	if err != nil {
		return nil, err
	}
	vmClient.Authorizer = authr
	vmClient.AddToUserAgent(userAgent)
	return &vmClient, nil
}

// CreateVolumeImage ...
func (a *Azure) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (a *Azure) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
