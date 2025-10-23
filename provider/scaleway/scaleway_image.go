package scaleway

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/scaleway/scaleway-sdk-go/api/block/v1"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"

	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// CustomizeImage returns image path with adaptations needed by cloud provider
func (h *Scaleway) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}

// BuildImage creates a Scaleway-compatible image from the active configuration.
func (h *Scaleway) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	c.Uefi = true

	if err := lepton.BuildImage(*c); err != nil {
		return "", err
	}

	return h.CustomizeImage(ctx)
}

// BuildImageWithPackage builds a Scaleway-compatible image that includes the provided package.
func (h *Scaleway) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	c.Uefi = true

	if err := lepton.BuildImageFromPackage(pkgpath, *c); err != nil {
		return "", err
	}
	return h.CustomizeImage(ctx)
}

// CreateImage uploads the image to object storage and registers a Scaleway snapshot.
func (h *Scaleway) CreateImage(ctx *lepton.Context, imagePath string) error {
	c := ctx.Config()

	h.ensureStorage()

	accessKeyID := os.Getenv("SCALEWAY_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("SCALEWAY_SECRET_ACCESS_KEY")
	orgID := os.Getenv("SCALEWAY_ORGANIZATION_ID")

	region := c.CloudConfig.Zone

	bucketName := c.CloudConfig.BucketName

	imageName := c.CloudConfig.ImageName
	newPath := c.CloudConfig.ImageName + ".qcow2"

	////// upload image

	cmd := exec.Command("sh", "-c", "qemu-img convert -f raw -O qcow2 ~/.ops/images/"+imageName+" ~/.ops/images/"+newPath)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Printf("%s\n", stdoutStderr)
	}

	// not sure this is actually necessary here; this just needs to ensure it's at least 1g
	cmd = exec.Command("sh", "-c", "qemu-img resize ~/.ops/images/"+newPath+" 1G")
	stdoutStderr, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Printf("%s\n", stdoutStderr)
	}

	opshome := lepton.GetOpsHome()

	sess := session.Must(session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		Endpoint:         aws.String(fmt.Sprintf("s3.%s.scw.cloud", region)),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
	}))

	svc := s3.New(sess)

	fileContent, err := ioutil.ReadFile(opshome + "/images/" + newPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return err
	}

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(newPath),
		Body:   bytes.NewReader(fileContent),
	})
	if err != nil {
		fmt.Printf("Error uploading object: %v\n", err)
		return err
	}

	blockAPI := block.NewAPI(h.client)

	//// create snapshot

	listSnapshotsResponse, err := blockAPI.ListSnapshots(&block.ListSnapshotsRequest{
		Zone: scw.ZonePlWaw1,
	})
	if err != nil {
		log.Fatalf("failed to list snapshots: %v", err)
	}

	fmt.Println("Existing Snapshots:")
	for _, snapshot := range listSnapshotsResponse.Snapshots {
		fmt.Printf("- ID: %s, Name: %s\n", snapshot.ID, snapshot.Name)
	}

	importedSnapshot, err := blockAPI.ImportSnapshotFromObjectStorage(&block.ImportSnapshotFromObjectStorageRequest{
		Zone:      scw.Zone(c.CloudConfig.Zone),
		Bucket:    bucketName,
		Key:       newPath,
		Name:      imageName,
		ProjectID: os.Getenv(orgID),
		Size:      scw.SizePtr(1 * 1024 * 1024 * 1024),
	})
	if err != nil {
		log.Fatalf("failed to import snapshot: %v", err)
	}

	fmt.Printf("Successfully imported snapshot: %s\n", importedSnapshot.ID)

	instanceAPI := instance.NewAPI(h.client)

	// need to sit && spin until status is 'ok'
	// this wait for looks like it returns a bit too early

	time.Sleep(10 * time.Second)
	fmt.Println("waiting..")
	res, err := instanceAPI.WaitForSnapshot(&instance.WaitForSnapshotRequest{
		SnapshotID: importedSnapshot.ID,
	})
	if err != nil {
		fmt.Printf("%+v", res)
	}

	/// create image

	snapshotID := importedSnapshot.ID
	imageArch := instance.ArchX86_64

	createImageReq := &instance.CreateImageRequest{
		Name:       imageName,
		Arch:       imageArch,
		Zone:       scw.Zone(c.CloudConfig.Zone),
		Project:    scw.StringPtr(orgID),
		RootVolume: snapshotID,
	}

	image, err := instanceAPI.CreateImage(createImageReq)
	if err != nil {
		log.Fatalf("Failed to create image from snapshot: %v", err)
	}

	fmt.Printf("%+v", image)

	return nil
}

// ListImages prints managed Scaleway snapshots using table or JSON output.
func (h *Scaleway) ListImages(ctx *lepton.Context, filter string) error {
	images, err := h.GetImages(ctx, filter)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID", "Status", "Size", "Created"})
	table.SetRowLine(true)

	for _, image := range images {
		size := ""
		if image.Size > 0 {
			size = lepton.Bytes2Human(image.Size)
		}
		row := []string{
			image.Name,
			image.ID,
			image.Status,
			size,
			lepton.Time2Human(image.Created),
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func (h *Scaleway) getImageByName(ctx *lepton.Context, imageName string) (*lepton.CloudImage, error) {
	var image *lepton.CloudImage

	images, err := h.GetImages(ctx, "")
	if err != nil {
		return image, err
	}

	for _, i := range images {
		if i.Name == imageName {
			image = &i
			return image, nil
		}
	}

	return image, errors.New("image not found")
}

// GetImages retrieves all managed Scaleway snapshots optionally filtered by name.
func (h *Scaleway) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
	images := []lepton.CloudImage{}

	instanceAPI := instance.NewAPI(h.client)

	listImagesRequest := &instance.ListImagesRequest{
		Zone:         scw.ZonePlWaw1,
		Organization: scw.StringPtr(os.Getenv("SCALEWAY_ORGANIZATION_ID")),
	}

	res, err := instanceAPI.ListImages(listImagesRequest)
	if err != nil {
		return images, err
	}

	for _, image := range res.Images {
		images = append(images, lepton.CloudImage{
			ID:     image.ID,
			Name:   image.Name,
			Status: string(image.State),
		})
	}

	return images, nil
}

// DeleteImage removes the Scaleway snapshot and associated object storage artifact.
func (h *Scaleway) DeleteImage(ctx *lepton.Context, imagename string) error {
	c := ctx.Config()

	instanceAPI := instance.NewAPI(h.client)

	i, err := h.getImageByName(ctx, imagename)
	if err != nil {
		return err
	}

	return instanceAPI.DeleteImage(&instance.DeleteImageRequest{
		Zone:    scw.Zone(c.CloudConfig.Zone),
		ImageID: i.ID,
	})

	return nil
}

// ResizeImage reports that resizing Scaleway snapshots is unsupported.
func (*Scaleway) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// SyncImage logs that Scaleway image synchronization is not implemented.
func (*Scaleway) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	log.Warn("not yet implemented")
	return nil
}
