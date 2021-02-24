package hyperv

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

var (
	vhdxImagesDir = lepton.GetOpsHome() + "/vhdx-images"
)

// BuildImage creates and converts a raw image to vhdx
func (p *Provider) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return p.createVhdxImage(c)
}

// BuildImageWithPackage creates and converts a raw image to vhdx using package image
func (p *Provider) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}

	return p.createVhdxImage(c)
}

func (p *Provider) createVhdxImage(c *types.Config) (imagePath string, err error) {

	vhdxImagesDir, err := findOrCreateHyperVImagesDir()
	if err != nil {
		return "", err
	}

	vhdxPath := path.Join(vhdxImagesDir, c.CloudConfig.ImageName+".vhdx")

	args := []string{
		"convert",
		"-O", "vhdx",
		c.RunConfig.Imagename, vhdxPath,
	}

	cmd := exec.Command("qemu-img", args...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return vhdxPath, nil
}

// CreateImage is a stub
func (p *Provider) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// ListImages prints hyper-v images in table format
func (p *Provider) ListImages(ctx *lepton.Context) error {
	images, err := p.GetImages(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Path", "Size", "CreatedAt"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, i := range images {
		var row []string
		row = append(row, i.Name)
		row = append(row, i.Path)
		row = append(row, lepton.Bytes2Human(i.Size))
		row = append(row, lepton.Time2Human(i.Created))
		table.Append(row)
	}
	table.Render()
	return nil
}

// GetImages returns the list of images available to run hyper-v virtual machines
func (p *Provider) GetImages(ctx *lepton.Context) (images []lepton.CloudImage, err error) {
	if _, err = os.Stat(vhdxImagesDir); os.IsNotExist(err) {
		return
	}

	err = filepath.Walk(vhdxImagesDir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()

		if len(name) > 4 && strings.Contains(info.Name(), ".vhdx") {
			images = append(images, lepton.CloudImage{
				Name:    strings.Replace(info.Name(), ".vhdx", "", 1),
				Path:    hostpath,
				Size:    info.Size(),
				Created: info.ModTime(),
			})
		}
		return nil
	})

	return
}

// DeleteImage removes hyper-v image
func (p *Provider) DeleteImage(ctx *lepton.Context, imagename string) error {
	imgpath := path.Join(vhdxImagesDir, imagename+".vhdx")
	err := os.Remove(imgpath)
	if err != nil {
		return err
	}
	return nil
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

func findOrCreateHyperVImagesDir() (string, error) {
	if _, err := os.Stat(vhdxImagesDir); os.IsNotExist(err) {
		os.MkdirAll(vhdxImagesDir, 0755)
	} else if err != nil {
		return "", err
	}

	return vhdxImagesDir, nil
}
