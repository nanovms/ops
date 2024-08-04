//go:build vbox || !onlyprovider

package vbox

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/nanovms/ops/wsl"
	"github.com/olekukonko/tablewriter"
)

var (
	vdiImagesDir = path.Join(lepton.GetOpsHome(), "vdi-images")
)

// BuildImage creates local image
func (p *Provider) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	imagePath, err := p.createVdiImage(c)
	if err != nil {
		return "", err
	}

	err = os.Remove(c.RunConfig.ImageName)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// BuildImageWithPackage creates local image using package image
func (p *Provider) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}

	imagePath, err := p.createVdiImage(c)
	if err != nil {
		return "", err
	}

	err = os.Remove(c.RunConfig.ImageName)
	if err != nil {
		return "", err
	}

	return imagePath, nil
}

// CreateImage is a stub
func (p *Provider) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// ListImages prints vcloud images in table format
func (p *Provider) ListImages(ctx *lepton.Context, filter string) error {
	images, err := p.GetImages(ctx, filter)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"UUID", "Name", "Status", "Size", "CreatedAt"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, i := range images {
		var row []string
		row = append(row, i.ID)
		row = append(row, i.Name)
		row = append(row, i.Status)
		row = append(row, lepton.Bytes2Human(i.Size))
		row = append(row, lepton.Time2Human(i.Created))
		table.Append(row)
	}
	table.Render()
	return nil
}

// GetImages returns the list of images available
func (p *Provider) GetImages(ctx *lepton.Context, filter string) (images []lepton.CloudImage, err error) {
	images = []lepton.CloudImage{}

	if _, err = os.Stat(vdiImagesDir); os.IsNotExist(err) {
		return
	}

	err = filepath.Walk(vdiImagesDir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()

		if len(name) > 4 && strings.Contains(info.Name(), ".vdi") {
			images = append(images, lepton.CloudImage{
				Name:    strings.Replace(info.Name(), ".vdi", "", 1),
				Path:    hostpath,
				Size:    info.Size(),
				Created: info.ModTime(),
			})
		}
		return nil
	})

	return
}

// DeleteImage removes VirtualBox image
func (p *Provider) DeleteImage(ctx *lepton.Context, imagename string) (err error) {

	imgpath := path.Join(vdiImagesDir, imagename+".vdi")

	if wsl.IsWSL() {
		imgpath, err = wsl.ConvertPathFromWSLtoWindows(imgpath)
		if err != nil {
			return
		}
	}

	cmd := exec.Command("VBoxManage", "closemedium", "disk", imgpath, "--delete")
	if err != nil {
		return
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New(string(output))
	}

	return
}

// ResizeImage is a stub
func (p *Provider) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return errors.New("Unsupported")
}

// SyncImage is a stub
func (p *Provider) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	return errors.New("Unsupported")
}

// CustomizeImage is a stub
func (p *Provider) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", errors.New("Unsupported")
}

func (p *Provider) createVdiImage(c *types.Config) (imagePath string, err error) {
	vdiImagesDir, err := findOrCreateVdiImagesDir()
	if err != nil {
		return
	}

	imagePath = path.Join(vdiImagesDir, c.CloudConfig.ImageName+".vdi")

	args := []string{
		"convert",
		"-O", "vdi",
		c.RunConfig.ImageName, imagePath,
	}

	cmd := exec.Command("qemu-img", args...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return
	}

	return
}

func findOrCreateVdiImagesDir() (string, error) {
	if _, err := os.Stat(vdiImagesDir); os.IsNotExist(err) {
		os.MkdirAll(vdiImagesDir, 0755)
	} else if err != nil {
		return "", err
	}

	return vdiImagesDir, nil
}
