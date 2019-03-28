package lepton

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// GCloudOperation status check
type GCloudOperation struct {
	service       *compute.Service
	projectID     string
	name          string
	area          string
	operationType string
}

func (gop *GCloudOperation) isDone(ctx context.Context) (bool, error) {
	var (
		op  *compute.Operation
		err error
	)
	fmt.Printf(".")
	switch gop.operationType {
	case "zone":
		op, err = gop.service.ZoneOperations.Get(gop.projectID, gop.area, gop.name).Context(ctx).Do()
	case "region":
		op, err = gop.service.RegionOperations.Get(gop.projectID, gop.area, gop.name).Context(ctx).Do()
	case "global":
		op, err = gop.service.GlobalOperations.Get(gop.projectID, gop.name).Context(ctx).Do()
	default:
		panic("We should never reach here")
	}
	if err != nil {
		return false, err
	}
	if op == nil || op.Status != "DONE" {
		return false, nil
	}
	if op.Error != nil && len(op.Error.Errors) > 0 && op.Error.Errors[0] != nil {
		e := op.Error.Errors[0]
		return false, fmt.Errorf("%v - %v", e.Code, e.Message)
	}

	return true, nil
}

// GCloud contains all operations for GCP
type GCloud struct {
	Storage *GCPStorage
}

func (p *GCloud) getArchiveName(ctx *Context) string {
	return ctx.config.CloudConfig.ImageName + ".tar.gz"
}

func (p *GCloud) pollOperation(ctx context.Context, context Context, service *compute.Service, op compute.Operation) error {
	var area, operationType string

	if strings.Contains(op.SelfLink, "zone") {
		s := strings.Split(op.Zone, "/")
		operationType = "zone"
		area = s[len(s)-1]
	} else if strings.Contains(op.SelfLink, "region") {
		s := strings.Split(op.Region, "/")
		operationType = "region"
		area = s[len(s)-1]
	} else {
		operationType = "global"
	}

	gOp := &GCloudOperation{
		service:       service,
		projectID:     context.config.CloudConfig.ProjectID,
		name:          op.Name,
		area:          area,
		operationType: operationType,
	}

	var pollCount int
	for {
		pollCount++

		status, err := gOp.isDone(ctx)
		if err != nil {
			fmt.Printf("Operation %s failed.\n", op.Name)
			return err
		}
		if status {
			break
		}
		// Wait for 120 seconds
		if pollCount > 60 {
			return fmt.Errorf("\nOperation timed out. No of tries %d", pollCount)
		}
		// TODO: Rate limit API instead of time.Sleep
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("\nOperation %s completed successfullly.\n", op.Name)
	return nil
}

// BuildImage to be upload on GCP
func (p *GCloud) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	imagePath := c.RunConfig.Imagename
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")

	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return "", fmt.Errorf("failed to unlink: %+v", err)
		}
	}

	err = os.Link(imagePath, symlink)
	if err != nil {
		return "", err
	}

	archPath := filepath.Join(filepath.Dir(imagePath), p.getArchiveName(ctx))
	files := []string{symlink}

	err = createArchive(archPath, files)
	if err != nil {
		return "", err
	}
	return archPath, nil
}

// Initialize GCP related things
func (p *GCloud) Initialize() error {
	p.Storage = &GCPStorage{}
	return nil
}

// CreateImage - Creates image on GCP using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *GCloud) CreateImage(ctx *Context) error {

	c := ctx.config
	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	sourceURL := fmt.Sprintf(GCPStorageURL,
		c.CloudConfig.BucketName, p.getArchiveName(ctx))

	rb := &compute.Image{
		Name: c.CloudConfig.ImageName,
		RawDisk: &compute.ImageRawDisk{
			Source: sourceURL,
		},
	}

	op, err := computeService.Images.Insert(c.CloudConfig.ProjectID, rb).Context(context).Do()
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	fmt.Printf("Image creation started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, *ctx, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Image creation succeeded %s.\n", c.CloudConfig.ImageName)
	return nil
}

// CreateInstance - Creates instance on Google Cloud Platform
func (p *GCloud) CreateInstance(ctx *Context) error {

	c := ctx.config
	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	// TODO: get from config
	machineType := "zones/us-west1-b/machineTypes/custom-1-2048"
	instanceName := fmt.Sprintf("%v-%v",
		filepath.Base(c.CloudConfig.ImageName),
		strconv.FormatInt(time.Now().Unix(), 10),
	)

	imageName := fmt.Sprintf("projects/%v/global/images/%v",
		c.CloudConfig.ProjectID,
		c.CloudConfig.ImageName)

	serialTrue := "true"

	rb := &compute.Instance{
		Name:        instanceName,
		MachineType: machineType,
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: imageName,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				Name: "eth0",
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						NetworkTier: "PREMIUM",
						Type:        "ONE_TO_ONE_NAT",
						Name:        "External NAT",
					},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   "serial-port-enable",
					Value: &serialTrue,
				},
			},
		},
		Tags: &compute.Tags{
			Items: []string{"http-server", "https-server"},
		},
	}
	// TODO : this always succeed, need to use self link for status.
	op, err := computeService.Instances.Insert(c.CloudConfig.ProjectID, "us-west1-b", rb).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation started using image %s. Monitoring operation %s.\n", imageName, op.Name)
	err = p.pollOperation(context, *ctx, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation succeeded %s.\n", instanceName)
	return nil
}

func createArchive(archive string, files []string) error {
	fd, err := os.Create(archive)
	if err != nil {
		return err
	}
	gzw := gzip.NewWriter(fd)

	tw := tar.NewWriter(gzw)

	for _, file := range files {
		fstat, err := os.Stat(file)
		if err != nil {
			return err
		}

		// write the header
		if err := tw.WriteHeader(&tar.Header{
			Name:   filepath.Base(file),
			Mode:   int64(fstat.Mode()),
			Size:   fstat.Size(),
			Format: tar.FormatGNU,
		}); err != nil {
			return err
		}

		fi, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data to tar
		if _, err := io.CopyN(tw, fi, fstat.Size()); err != nil {
			return err
		}
		if err = fi.Close(); err != nil {
			return err
		}
	}

	// Explicitly close all writers in correct order without any error
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gzw.Close(); err != nil {
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	return nil
}
