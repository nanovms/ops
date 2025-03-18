//go:build gcp || !onlyprovider

package gcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

	compute "google.golang.org/api/compute/v1"
)

// CreateVolume creates local volume and converts it to GCP format before orchestrating the necessary upload procedures
func (g *GCloud) CreateVolume(ctx *lepton.Context, cv types.CloudVolume, data string, provider string) (lepton.NanosVolume, error) {
	config := ctx.Config()

	var sizeInGb int64
	var vol lepton.NanosVolume
	if config.BaseVolumeSz != "" {
		size, err := lepton.GetSizeInGb(config.BaseVolumeSz)
		if err != nil {
			return vol, fmt.Errorf("cannot get volume size: %v", err)
		}
		config.BaseVolumeSz = "" // create minimum-sized local volume
		sizeInGb = int64(size)
	}

	lv, err := lepton.CreateLocalVolume(config, cv.Name, data, provider)
	if err != nil {
		return lv, err
	}
	defer os.Remove(lv.Path)

	arch := cv.Name + ".tar.gz"
	archPath := filepath.Join(filepath.Dir(lv.Path), arch)

	files := map[string]string{
		lv.Path: "disk.raw",
	}
	err = lepton.CreateArchive(archPath, files)
	if err != nil {
		return lv, err
	}
	defer os.Remove(archPath)

	err = g.Storage.CopyToBucket(config, archPath)
	if err != nil {
		return lv, err
	}
	defer g.Storage.DeleteFromBucket(config, filepath.Base(archPath))

	labels := buildGcpLabels(nil, "")
	img := &compute.Image{
		Name:   cv.Name,
		Labels: labels,
		RawDisk: &compute.ImageRawDisk{
			Source: fmt.Sprintf(GCPStorageURL, config.CloudConfig.BucketName, arch),
		},
	}
	op, err := g.Service.Images.Insert(config.CloudConfig.ProjectID, img).Context(context.TODO()).Do()
	if err != nil {
		return lv, err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return lv, err
	}

	disk := &compute.Disk{
		Name:        cv.Name,
		Labels:      labels,
		SizeGb:      sizeInGb,
		SourceImage: "global/images/" + cv.Name,
		Type:        fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", config.CloudConfig.ProjectID, config.CloudConfig.Zone),
	}

	op, err = g.Service.Disks.Insert(config.CloudConfig.ProjectID, config.CloudConfig.Zone, disk).Context(context.TODO()).Do()
	if err != nil {
		return lv, err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return lv, err
	}
	return lv, nil
}

// GetAllVolumes gets all volumes created in GCP as Compute Engine Disks
func (g *GCloud) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	config := ctx.Config()

	var volumes []lepton.NanosVolume

	projectID := config.CloudConfig.ProjectID
	if strings.Compare(projectID, "") == 0 {
		return nil, errGCloudProjectIDMissing()
	}

	zone := config.CloudConfig.Zone
	if strings.Compare(zone, "") == 0 {
		return nil, errGCloudZoneMissing()
	}

	dl, err := g.Service.Disks.List(projectID, zone).Context(context.TODO()).Do()
	if err != nil {
		return nil, err
	}

	for _, d := range dl.Items {
		var users []string
		for _, u := range d.Users {
			uri := strings.Split(u, "/")
			users = append(users, uri[len(uri)-1])
		}

		vol := lepton.NanosVolume{
			ID:         strconv.Itoa(int(d.Id)),
			Name:       d.Name,
			Status:     d.Status,
			Size:       strconv.Itoa(int(d.SizeGb)),
			Path:       d.SelfLink,
			CreatedAt:  d.CreationTimestamp,
			AttachedTo: strings.Join(users, ";"),
		}

		volumes = append(volumes, vol)
	}

	return &volumes, nil
}

// DeleteVolume deletes specific disk and image in GCP
func (g *GCloud) DeleteVolume(ctx *lepton.Context, name string) error {
	config := ctx.Config()

	op, err := g.Service.Disks.Delete(config.CloudConfig.ProjectID, config.CloudConfig.Zone, name).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}

	op, err = g.Service.Images.Delete(config.CloudConfig.ProjectID, name).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches Compute Engine Disk volume to existing instance
func (g *GCloud) AttachVolume(ctx *lepton.Context, image, name string, attachID int) error {
	config := ctx.Config()

	disk := &compute.AttachedDisk{
		AutoDelete: false,
		Source:     fmt.Sprintf("zones/%s/disks/%s", config.CloudConfig.Zone, name),
	}
	if attachID >= 0 {
		if attachID > 0 {
			disk.DeviceName = fmt.Sprintf("persistent-disk-%d", attachID)
		} else {
			return fmt.Errorf("attachment point 0 is reserved for the boot disk")
		}
	}
	op, err := g.Service.Instances.AttachDisk(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image, disk).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}

	err = g.ResetInstance(lepton.NewContext(config), image)
	if err != nil {
		return fmt.Errorf("error reseting instance %v: %v", image, err)
	}

	return nil
}

// DetachVolume detaches Compute Engine Disk volume from existing instance
func (g *GCloud) DetachVolume(ctx *lepton.Context, image, volumeName string) error {
	config := ctx.Config()

	var mount string

	ins, err := g.Service.Instances.Get(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	for _, d := range ins.Disks {
		name := strings.Split(d.Source, "/")
		if volumeName == name[len(name)-1] {
			mount = d.DeviceName
			break
		}
	}
	if mount == "" {
		return fmt.Errorf("volume %s not found in %s", volumeName, image)
	}
	op, err := g.Service.Instances.DetachDisk(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image, mount).Context(context.TODO()).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(context.TODO(), config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}

	err = g.ResetInstance(lepton.NewContext(config), image)
	if err != nil {
		return fmt.Errorf("error reseting instance %v: %v", image, err)
	}

	return nil
}
