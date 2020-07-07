package lepton

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureStorage provides Azure storage related operations
type AzureStorage struct{}

type qemuInfo struct {
	VirtualSize uint32 `json:"virtual-size"`
	Filename    string `json:"filename"`
	Format      string `json:"format"`
	ActualSize  uint32 `json:"actual-size"`
	DirtyFlag   bool   `json:"dirty-flag"`
}

const (
	onemb = 1048576
)

func roundup(x, y uint32) uint32 {
	n := (x + y - 1) / y
	return (n * onemb)
}

func (az *AzureStorage) resizeLength(virtSz uint32) uint32 {
	var azureMin uint32 = 20971520 // min disk sz
	var max uint32

	if azureMin > virtSz {
		max = azureMin
	} else {
		max = virtSz
	}

	return roundup(max, onemb)
}

// might have to adjust this if disk sz is really large/overflows
func (az *AzureStorage) virtualSize(archPath string) uint32 {
	args := []string{
		"info", "-f", "raw",
		"--output", "json", archPath,
	}

	cmd := exec.Command("qemu-img", args...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}

	qi := &qemuInfo{}
	err = json.Unmarshal([]byte(out), qi)

	return qi.VirtualSize
}

func (az *AzureStorage) resizeImage(basePath string, newPath string, resizeSz uint32) {
	in, err := os.Open(basePath)
	if err != nil {
		fmt.Println(err)
	}
	defer in.Close()

	out, err := os.Create(newPath)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		fmt.Println(err)
	}

	szstr := fmt.Sprint(resizeSz)

	args := []string{
		"resize", "-f", "raw",
		newPath, szstr,
	}

	cmd := exec.Command("qemu-img", args...)
	_, err = cmd.Output()
	if err != nil {
		fmt.Println(err)
	}
}

// CopyToBucket copies archive to bucket
func (az *AzureStorage) CopyToBucket(config *Config, archPath string) error {

	// not sure why this is necessary - afaik only gcp does the tarball
	// uploads
	base := config.CloudConfig.ImageName + ".img"
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", base)
	imgpath = strings.ReplaceAll(imgpath, "-image", "")

	// get virtual size
	vs := az.virtualSize(imgpath)
	rs := az.resizeLength(vs)

	debug := false
	if debug {
		fmt.Printf("virt sz: %d\n", vs)
		fmt.Printf("resize sz: %d\n", rs)
	}

	newpath := "/tmp/" + base
	newpath = strings.ReplaceAll(newpath, "-image", "")

	// resize
	az.resizeImage(imgpath, newpath, rs)

	// convert
	vhdPath := "/tmp/" + config.CloudConfig.ImageName + ".vhd"
	vhdPath = strings.ReplaceAll(vhdPath, "-image", "")

	// this is probably just for hyper-v not azure
	args := []string{
		"convert", "-f", "raw",
		"-O", "vpc", "-o", "subformat=fixed,force_size",
		newpath, vhdPath,
	}

	cmd := exec.Command("qemu-img", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}

	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		log.Fatal("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	containerName := "quickstart-nanos"

	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	containerURL := azblob.NewContainerURL(*URL, p)

	// we can skip over this if it already exists
	fmt.Printf("Creating a container named %s\n", containerName)
	ctx := context.Background()
	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		fmt.Println(err)
	}

	blobURL := containerURL.NewBlockBlobURL(config.CloudConfig.ImageName + ".vhd")

	file, err := os.Open(vhdPath)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Uploading the file with blob name: %s\n", vhdPath)
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (az *AzureStorage) DeleteFromBucket(config *Config, key string) error {

	fmt.Println("un-implemented")

	return nil
}
