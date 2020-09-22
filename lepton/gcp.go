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

	"github.com/olekukonko/tablewriter"

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

func checkCredentialsProvided() error {
	creds, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if !ok {
		return fmt.Errorf(ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\nFollow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
	}
	if _, err := os.Stat(creds); os.IsNotExist(err) {
		return fmt.Errorf(ErrorColor, fmt.Sprintf("error: File %s mentioned in GOOGLE_APPLICATION_CREDENTIALS does not exist.", creds))
	}
	return nil
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

func (p *GCloud) pollOperation(ctx context.Context, projectID string, service *compute.Service, op compute.Operation) error {
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
		projectID:     projectID,
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
	fmt.Printf("\nOperation %s completed successfully.\n", op.Name)
	return nil
}

func (p *GCloud) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")

	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return "", fmt.Errorf("failed to unlink: %+v", err)
		}
	}

	err := os.Link(imagePath, symlink)
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

// BuildImage to be upload on GCP
func (p *GCloud) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.customizeImage(ctx)
}

// BuildImageWithPackage to upload on GCP
func (p *GCloud) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.customizeImage(ctx)
}

// Initialize GCP related things
func (p *GCloud) Initialize() error {
	p.Storage = &GCPStorage{}
	return nil
}

// CreateImage - Creates image on GCP using nanos images
// TODO : re-use and cache DefaultClient and instances.
func (p *GCloud) CreateImage(ctx *Context) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}

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
	err = p.pollOperation(context, c.CloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Image creation succeeded %s.\n", c.CloudConfig.ImageName)
	return nil
}

// ListImages lists images on Google Cloud
func (p *GCloud) ListImages(ctx *Context) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}
	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return err
	}
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	req := computeService.Images.List(creds.ProjectID)
	if err := req.Pages(context, func(page *compute.ImageList) error {
		for _, image := range page.Items {
			var row []string
			row = append(row, image.Name)
			row = append(row, fmt.Sprintf("%v", image.Status))
			row = append(row, image.CreationTimestamp)
			table.Append(row)
		}
		return nil
	}); err != nil {
		return err
	}
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	table.Render()
	return nil
}

// DeleteImage deletes image from Gcloud
func (p *GCloud) DeleteImage(ctx *Context, imagename string) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}
	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return err
	}
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return err
	}
	op, err := computeService.Images.Delete(creds.ProjectID, imagename).Context(context).Do()
	if err != nil {
		return fmt.Errorf("error:%+v", err)
	}
	err = p.pollOperation(context, creds.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Image deletion succeeded %s.\n", imagename)
	return nil
}

// CreateInstance - Creates instance on Google Cloud Platform
func (p *GCloud) CreateInstance(ctx *Context) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}

	context := context.TODO()
	creds, err := google.FindDefaultCredentials(context)
	if err != nil {
		return err
	}

	c := ctx.config
	if c.CloudConfig.Zone == "" {
		return fmt.Errorf("Zone not provided in config.CloudConfig")
	}

	if c.CloudConfig.ProjectID == "" {
		fmt.Printf("ProjectId not provided in config.CloudConfig. Using %s from default credentials.", creds.ProjectID)
		c.CloudConfig.ProjectID = creds.ProjectID
	}

	if c.CloudConfig.Flavor == "" {
		return fmt.Errorf("Flavor not provided in config.CloudConfig")
	}

	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	machineType := fmt.Sprintf("zones/%s/machineTypes/%s", c.CloudConfig.Zone, c.CloudConfig.Flavor)
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
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: imageName,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Name: "eth0",
				AccessConfigs: []*compute.AccessConfig{
					{
						NetworkTier: "PREMIUM",
						Type:        "ONE_TO_ONE_NAT",
						Name:        "External NAT",
					},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "serial-port-enable",
					Value: &serialTrue,
				},
			},
		},
		Tags: &compute.Tags{
			Items: []string{"http-server", "https-server"},
		},
	}
	op, err := computeService.Instances.Insert(c.CloudConfig.ProjectID, c.CloudConfig.Zone, rb).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation started using image %s. Monitoring operation %s.\n", imageName, op.Name)
	err = p.pollOperation(context, c.CloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Instance creation succeeded %s.\n", instanceName)
	return nil
}

// ListInstances lists instances on Gcloud
func (p *GCloud) ListInstances(ctx *Context) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}

	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	req := computeService.Instances.List(ctx.config.CloudConfig.ProjectID, ctx.config.CloudConfig.Zone)
	if err := req.Pages(context, func(page *compute.InstanceList) error {
		for _, instance := range page.Items {

			var rows []string
			rows = append(rows, instance.Name)
			rows = append(rows, instance.Status)
			rows = append(rows, instance.CreationTimestamp)

			var privateIps, publicIps []string
			for _, ninterface := range instance.NetworkInterfaces {
				if ninterface.NetworkIP != "" {
					privateIps = append(privateIps, ninterface.NetworkIP)

				}
				for _, accessConfig := range ninterface.AccessConfigs {
					if accessConfig.NatIP != "" {
						publicIps = append(publicIps, accessConfig.NatIP)
					}
				}
			}
			rows = append(rows, strings.Join(privateIps, ","))
			rows = append(rows, strings.Join(publicIps, ","))
			table.Append(rows)
		}
		return nil
	}); err != nil {
		return err
	}
	table.Render()
	return nil
}

// GetInstances return all instances on GCloud
func (p *GCloud) GetInstances(ctx *Context) ([]compute.Instance, error) {
	if err := checkCredentialsProvided(); err != nil {
		return nil, err
	}

	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	var (
		instances []compute.Instance
		req       = computeService.Instances.List(ctx.config.CloudConfig.ProjectID, ctx.config.CloudConfig.Zone)
	)

	if err := req.Pages(context, func(page *compute.InstanceList) error {
		for i := range page.Items {
			instances = append(instances, *page.Items[i])
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return instances, nil
}

// DeleteInstance deletes instance from Gcloud
func (p *GCloud) DeleteInstance(ctx *Context, instancename string) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}
	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return err
	}
	cloudConfig := ctx.config.CloudConfig
	op, err := computeService.Instances.Delete(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Instance deletion started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}
	fmt.Printf("Instance deletion succeeded %s.\n", instancename)
	return nil
}

// StartInstance starts an instance in GCloud
func (p *GCloud) StartInstance(ctx *Context, instancename string) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}

	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	cloudConfig := ctx.config.CloudConfig
	op, err := computeService.Instances.Start(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}

	fmt.Printf("Instance started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Instance started %s.\n", instancename)
	return nil

}

// StopInstance deletes instance from GCloud
func (p *GCloud) StopInstance(ctx *Context, instancename string) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}

	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}

	computeService, err := compute.New(client)
	if err != nil {
		return err
	}

	cloudConfig := ctx.config.CloudConfig
	op, err := computeService.Instances.Stop(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Context(context).Do()
	if err != nil {
		return err
	}

	fmt.Printf("Instance stopping started. Monitoring operation %s.\n", op.Name)
	err = p.pollOperation(context, cloudConfig.ProjectID, computeService, *op)
	if err != nil {
		return err
	}

	fmt.Printf("Instance stop succeeded %s.\n", instancename)
	return nil
}

// GetInstanceLogs gets instance related logs
func (p *GCloud) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {
	if err := checkCredentialsProvided(); err != nil {
		return err
	}
	context := context.TODO()
	client, err := google.DefaultClient(context, compute.CloudPlatformScope)
	if err != nil {
		return err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return err
	}
	cloudConfig := ctx.config.CloudConfig
	lastPos := int64(0)
	for {
		resp, err := computeService.Instances.GetSerialPortOutput(cloudConfig.ProjectID, cloudConfig.Zone, instancename).Start(lastPos).Context(context).Do()
		if err != nil {
			return err
		}
		if resp.Contents != "" {
			fmt.Printf("%s", resp.Contents)
		}
		if lastPos == resp.Next && !watch {
			break
		}
		lastPos = resp.Next
		time.Sleep(time.Second)
	}
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

// ResizeImage is not supported on google cloud.
func (p *GCloud) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}
