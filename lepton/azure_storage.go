package lepton

import (
	"bytes"
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
	onemb         = 1048576
	containerName = "quickstart-nanos"
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

	ctx := context.Background()
	containerURL := getContainerURL(containerName)

	if !containerExists(containerURL) {
		fmt.Printf("Creating a container named %s\n", containerName)
		_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
		if err != nil {
			fmt.Println(err)
		}
	}

	blobURL := containerURL.NewPageBlobURL(config.CloudConfig.ImageName + ".vhd")

	file, err := os.Open(vhdPath)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		fmt.Println(err)
	}

	max := 4194304

	length := fi.Size()
	ilength := int(length)
	q, r := int(ilength/max), ilength%max
	if r != 0 {
		q++
	}

	_, err = blobURL.Create(ctx, length, 0, azblob.BlobHTTPHeaders{},
		azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < q; i++ {
		page := make([]byte, max)
		n, err := file.Read(page)

		_, err = blobURL.UploadPages(ctx, int64(i*max), bytes.NewReader(page[:n]), azblob.PageBlobAccessConditions{}, nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	return nil
}

// DeleteFromBucket deletes key from config's bucket
func (az *AzureStorage) DeleteFromBucket(config *Config, key string) error {

	fmt.Printf("Started deleting image from container")
	blobUrl := getBlobURL(containerName, key)

	ctx := context.Background()
	_, err := blobUrl.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})

	if err != nil {
		return err
	}

	return nil
}

//Exists() function not available in sdk. So for now this is a work around
func containerExists(containerUrl azblob.ContainerURL) bool {
	_, err := containerUrl.GetProperties(context.Background(), azblob.LeaseAccessConditions{})
	if err != nil {
		return false
	}
	return true
}

// return AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_ACCESS_KEY
func getContainerURL(containerName string) azblob.ContainerURL {
	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		exitWithError("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		exitWithError("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	containerURL := azblob.NewContainerURL(*URL, p)

	return containerURL
}

func getBlobURL(container string, blobname string) azblob.BlobURL {

	containerURL := getContainerURL(container)

	return containerURL.NewBlobURL(blobname)

}
