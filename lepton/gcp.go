package lepton

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

type GCloud struct {
	Storage *GCPStorage
}

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
	} else if os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check symlink: %+v", err)
	}

	err = os.Symlink(imagePath, symlink)
	if err != nil {
		return "", err
	}

	archPath := filepath.Join(filepath.Dir(imagePath), c.CloudConfig.ArchiveName)
	files := []string{symlink}

	err = createArchive(archPath, files)
	if err != nil {
		return "", err
	}
	return archPath, nil
}

func (p *GCloud) Initialize() error {
	p.Storage = &GCPStorage{}
	return nil
}

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
		c.CloudConfig.BucketName, c.CloudConfig.ArchiveName)

	rb := &compute.Image{
		Name: strings.Split(c.CloudConfig.ArchiveName, ".")[0],
		RawDisk: &compute.ImageRawDisk{
			Source: sourceURL,
		},
	}

	// TODO : This always succeed, need to use the selflink in response to
	// check for status.
	op, err := computeService.Images.Insert(c.CloudConfig.ProjectID, rb).Context(context).Do()
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	fmt.Printf("Image creation started. check %s for status\n", op.SelfLink)
	return nil
}

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
	instanceName := filepath.Base(c.Program) + "_1"

	imageName := fmt.Sprintf("projects/%v/global/images/%v",
		c.CloudConfig.ProjectID,
		strings.Split(c.CloudConfig.ArchiveName, ".")[0])

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
			&compute.NetworkInterface{Name: "eth0"},
		},
	}
	// TODO : this always succeed, need to use self link for status.
	op, err := computeService.Instances.Insert(c.CloudConfig.ProjectID, "us-west1-b", rb).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation started. check %s for status\n", op.SelfLink)
	return nil
}

func createArchive(archive string, files []string) error {
	fd, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer fd.Close()
	gzw := gzip.NewWriter(fd)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(stat, stat.Name())
		if err != nil {
			return err
		}
		// update the name to correctly
		header.Name = filepath.Base(file)
		header.Format = tar.FormatGNU

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		fi, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data to tar
		if _, err := io.Copy(tw, fi); err != nil {
			return err
		}
		fi.Close()
	}
	return nil
}
