package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"os"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// BuildImage to be upload on v
func (p *ProxMox) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on ProxMox.
func (p *ProxMox) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return p.CustomizeImage(ctx)
}

func (p *ProxMox) createImage(key string, bucket string, region string) {
}

func (p *ProxMox) destroyImage(snapshotid string) {
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		pwd, _ := os.Getwd()
		fmt.Println("PWD: ", pwd)
		fmt.Println(err)
		os.Exit(1)
	}
	return r
}

// CreateImage - Creates image on v using nanos images
func (p *ProxMox) CreateImage(ctx *lepton.Context, imagePath string) error {

	var err error

	p.imageName = ctx.Config().TargetConfig["ImageName"]
	p.isoStorageName = ctx.Config().TargetConfig["IsoStorageName"]

	if p.isoStorageName == "" {
		p.isoStorageName = "local"
	}

	err = p.CheckStorage(p.isoStorageName, "iso")
	if err != nil {
		return err
	}

	opshome := lepton.GetOpsHome()

	fieldName := "filename"
	fileName := opshome + "/images/" + p.imageName

	var b bytes.Buffer

	w := multipart.NewWriter(&b)

	var fw io.Writer

	err = w.WriteField("content", "iso")
	if err != nil {
		fmt.Println(err)
		return err
	}

	file := mustOpen(fileName)

	fw, err = w.CreateFormFile(fieldName, file.Name()+".iso")
	if err != nil {
		fmt.Printf("Error creating writer: %v\n", err)
		return err
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		fmt.Printf("Error with io.Copy: %v\n", err)
		return err
	}

	w.Close()

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/storage/"+p.isoStorageName+"/upload", &b)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "createimage", p.isoStorageName)
	if err != nil {
		return err
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	return nil
}

// GetImages return all images on ProxMox
func (p *ProxMox) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}

// ImageResponse contains a list of Image info structs.
type ImageResponse struct {
	Data []ImageInfo `json:"data"`
}

// ImageInfo contains information on the uploaded disk images.
type ImageInfo struct {
	Volid string `json:"volid"`
	Size  int    `json:"size"`
}

// ListImages lists images on ProxMox
func (p *ProxMox) ListImages(ctx *lepton.Context) error {

	var err error

	p.isoStorageName = ctx.Config().TargetConfig["IsoStorageName"]

	if p.isoStorageName == "" {
		p.isoStorageName = "local"
	}

	err = p.CheckStorage(p.isoStorageName, "iso")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/storage/"+p.isoStorageName+"/content", nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "listimages", p.isoStorageName)
	if err != nil {
		return err
	}

	ir := &ImageResponse{}
	json.Unmarshal([]byte(body), ir)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Size"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range ir.Data {
		o := strings.Split(image.Volid, "/")
		var row []string
		row = append(row, image.Volid)
		row = append(row, o[1])
		row = append(row, strconv.Itoa(image.Size))
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from v
func (p *ProxMox) DeleteImage(ctx *lepton.Context, snapshotID string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (p *ProxMox) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on ProxMox.
func (p *ProxMox) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (p *ProxMox) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.Imagename
	return imagePath, nil
}
