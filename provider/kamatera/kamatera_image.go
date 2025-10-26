package kamatera

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// CustomizeImage returns image path with adaptations needed by cloud provider
func (h *Kamatera) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}

// BuildImage creates a Kamatera-compatible image from the active configuration.
func (h *Kamatera) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	c.Uefi = true

	if err := lepton.BuildImage(*c); err != nil {
		return "", err
	}

	return h.CustomizeImage(ctx)
}

// BuildImageWithPackage builds a Kamatera-compatible image that includes the provided package.
func (h *Kamatera) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	c.Uefi = true

	if err := lepton.BuildImageFromPackage(pkgpath, *c); err != nil {
		return "", err
	}
	return h.CustomizeImage(ctx)
}

// CreateImage uploads the image to object storage and registers a Kamatera snapshot.
func (h *Kamatera) CreateImage(ctx *lepton.Context, imagePath string) error {
	c := ctx.Config()

	h.ensureStorage()

	region := c.CloudConfig.Zone

	bucketName := c.CloudConfig.BucketName

	imageName := c.CloudConfig.ImageName
	newPath := c.CloudConfig.ImageName + ".qcow2"

	bucketId := os.Getenv("bucket_id")
	bucketAccess := os.Getenv("bucket_access_key")
	bucketSecret := os.Getenv("bucket_secret")

	fmt.Printf(bucketId)

	/*
	   https://store-us-sc.images.cloudwm.com
	   Bucket ID:9330e629-b704-4fd7-9244-67976d9213a6
	   Access Key:21c4e1d6-0b4f-4006-8c95-757f3e8f16c1
	   Secret Key:e7693a08-0933-4e28-a5d5-caa6dfb6ea94
	*/

	////// upload image

	cmd := exec.Command("sh", "-c", "qemu-img convert -f raw -O qcow2 ~/.ops/images/"+imageName+" ~/.ops/images/"+newPath)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Printf("%s\n", stdoutStderr)
	}

	opshome := lepton.GetOpsHome()

	sess := session.Must(session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(bucketAccess, bucketSecret, ""),
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

	//fmt.Printf("%+v", image)

	return nil
}

// ListImages prints managed Kamatera snapshots using table or JSON output.
func (h *Kamatera) ListImages(ctx *lepton.Context, filter string) error {
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

func (h *Kamatera) getImageByName(ctx *lepton.Context, imageName string) (*lepton.CloudImage, error) {
	var image *lepton.CloudImage

	images, err := h.GetImages(ctx, "")
	if err != nil {
		return nil, err
	}

	for _, i := range images {
		if i.Name == imageName {
			image = &i
			return image, nil
		}
	}

	return image, errors.New("image not found")
}

// GetImages retrieves all managed Kamatera snapshots optionally filtered by name.
func (h *Kamatera) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
	//	c := ctx.Config()
	images := []lepton.CloudImage{}

	url := "https://console.kamatera.com/svc/hdlib?listMode=1&auxFilter=US-SC&filter=&size=10&sorting=name&from="

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println(h.apiKey)

	req.Header.Add("api-version", "latest")
	req.Header.Add("Authorization", "Bearer "+h.apiKey)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Println(string(body))

	/*	for _, image := range res.Images {
			images = append(images, lepton.CloudImage{
				ID:     image.ID,
				Name:   image.Name,
				Status: string(image.State),
			})
		}
	*/

	return images, nil
}

// DeleteImage removes the Kamatera snapshot and associated object storage artifact.
func (h *Kamatera) DeleteImage(ctx *lepton.Context, imagename string) error {
	//	c := ctx.Config()

	return nil
}

// ResizeImage reports that resizing Kamatera snapshots is unsupported.
func (*Kamatera) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// SyncImage logs that Kamatera image synchronization is not implemented.
func (*Kamatera) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	log.Warn("not yet implemented")
	return nil
}
