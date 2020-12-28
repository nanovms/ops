package lepton

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// BuildImage for onprem
func (p *OnPrem) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	return "", err
}

// BuildImageWithPackage for onprem
func (p *OnPrem) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return "", nil
}

// CreateImage on prem
// assumes local for now
func (p *OnPrem) CreateImage(ctx *Context, imagePath string) error {
	// this method implementation is not necessary as BuildImage and BuildImageWithPackage creates an image locally
	return nil
}

// ResizeImage resizes the lcoal image imagename. You should never
// specify a negative size.
func (p *OnPrem) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", imagename)

	bytes, err := parseBytes(hbytes)
	if err != nil {
		return err
	}

	return os.Truncate(imgpath, bytes)
}

// GetImages return all images on prem
func (p *OnPrem) GetImages(ctx *Context) ([]CloudImage, error) {
	return nil, errors.New("un-implemented")
}

// ListImages on premise
func (p *OnPrem) ListImages(ctx *Context) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images")

	if _, err := os.Stat(imgpath); os.IsNotExist(err) {
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
	err := filepath.Walk(imgpath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()

		if len(name) > 4 && strings.LastIndex(info.Name(), ".img") == len(name)-4 {
			var row []string
			row = append(row, info.Name())
			row = append(row, hostpath)
			row = append(row, bytes2Human(info.Size()))
			row = append(row, time2Human(info.ModTime()))
			table.Append(row)
		}
		return nil
	})
	if err != nil {
		return err
	}
	table.Render()
	return nil
}

// DeleteImage on premise
func (p *OnPrem) DeleteImage(ctx *Context, imagename string) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", imagename)
	err := os.Remove(imgpath)
	if err != nil {
		return err
	}
	return nil
}

// SyncImage syncs image from onprem to target provider provided in Context
func (p *OnPrem) SyncImage(config *Config, target Provider, image string) error {
	imagePath := path.Join(localImageDir, image+".img")
	_, err := os.Stat(imagePath)
	if err != nil {
		return nil
	}
	config.RunConfig.Imagename = imagePath
	config.CloudConfig.ImageName = image

	// customizes image for target
	ctx := NewContext(config)
	archive, err := target.CustomizeImage(ctx)
	if err != nil {
		return err
	}

	return target.CreateImage(ctx, archive)
}

// CustomizeImage for onprem as stub to satisfy interface
func (p *OnPrem) CustomizeImage(ctx *Context) (string, error) {
	return "", nil
}
