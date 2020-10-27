package lepton

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	compute "google.golang.org/api/compute/v1"
)

// CreateVolume creates local volume and converts it to GCP format before orchestrating the necessary upload procedures
func (g *GCloud) CreateVolume(config *Config, name, data, size, provider string) (NanosVolume, error) {
	arch := name + ".tar.gz"
	ctx := context.Background()

	lv, err := CreateLocalVolume(config, name, data, size, provider)
	if err != nil {
		return lv, err
	}

	link := filepath.Join(filepath.Dir(lv.Path), "disk.raw")
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return lv, fmt.Errorf("failed to unlink: %+v", err)
		}
	}
	err = os.Link(lv.Path, link)
	if err != nil {
		return lv, err
	}
	archPath := filepath.Join(filepath.Dir(lv.Path), arch)
	// compress it into a tar.gz file
	err = createArchive(archPath, []string{link})
	if err != nil {
		return lv, err
	}

	err = g.Storage.CopyToBucket(config, archPath)
	if err != nil {
		return lv, err
	}

	img := &compute.Image{
		Name: name,
		RawDisk: &compute.ImageRawDisk{
			Source: fmt.Sprintf(GCPStorageURL, config.CloudConfig.BucketName, arch),
		},
	}
	op, err := g.Service.Images.Insert(config.CloudConfig.ProjectID, img).Context(ctx).Do()
	if err != nil {
		return lv, err
	}
	err = g.pollOperation(ctx, config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return lv, err
	}

	disk := &compute.Disk{
		Name:        name,
		SourceImage: "global/images/" + name,
		Type:        fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", config.CloudConfig.ProjectID, config.CloudConfig.Zone),
	}

	_, err = g.Service.Disks.Insert(config.CloudConfig.ProjectID, config.CloudConfig.Zone, disk).Context(ctx).Do()
	if err != nil {
		return lv, err
	}
	return lv, nil
}

// GetAllVolumes gets all volumes created in GCP as Compute Engine Disks
func (g *GCloud) GetAllVolumes(config *Config) (*[]NanosVolume, error) {
	var volumes []NanosVolume
	ctx := context.Background()

	projectID := config.CloudConfig.ProjectID
	if strings.Compare(projectID, "") == 0 {
		return nil, errGCloudProjectIDMissing()
	}

	zone := config.CloudConfig.Zone
	if strings.Compare(zone, "") == 0 {
		return nil, errGCloudZoneMissing()
	}

	dl, err := g.Service.Disks.List(projectID, zone).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	for _, d := range dl.Items {
		var users []string
		for _, u := range d.Users {
			uri := strings.Split(u, "/")
			users = append(users, uri[len(uri)-1])
		}

		vol := NanosVolume{
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
func (g *GCloud) DeleteVolume(config *Config, name string) error {
	ctx := context.Background()

	_, err := g.Service.Disks.Delete(config.CloudConfig.ProjectID, config.CloudConfig.Zone, name).Context(ctx).Do()
	if err != nil {
		return err
	}

	_, err = g.Service.Images.Delete(config.CloudConfig.ProjectID, name).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches Compute Engine Disk volume to existing instance
func (g *GCloud) AttachVolume(config *Config, image, name, mount string) error {
	ctx := context.Background()
	disk := &compute.AttachedDisk{
		AutoDelete: false,
		DeviceName: mount,
		Source:     fmt.Sprintf("zones/%s/disks/%s", config.CloudConfig.Zone, name),
	}
	op, err := g.Service.Instances.AttachDisk(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image, disk).Context(ctx).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(ctx, config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}
	return nil
}

// DetachVolume detaches Compute Engine Disk volume from existing instance
func (g *GCloud) DetachVolume(config *Config, image, volumeName string) error {
	var mount string
	ctx := context.Background()
	ins, err := g.Service.Instances.Get(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image).Context(ctx).Do()
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
	op, err := g.Service.Instances.DetachDisk(config.CloudConfig.ProjectID, config.CloudConfig.Zone, image, mount).Context(ctx).Do()
	if err != nil {
		return err
	}
	err = g.pollOperation(ctx, config.CloudConfig.ProjectID, g.Service, *op)
	if err != nil {
		return err
	}
	return nil
}
