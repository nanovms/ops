package lepton

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nanovms/ops/config"
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
func (p *OnPrem) GetImages(ctx *Context) (images []CloudImage, err error) {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images")

	if _, err = os.Stat(imgpath); os.IsNotExist(err) {
		return
	}

	err = filepath.Walk(imgpath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()

		if len(name) > 4 && strings.LastIndex(info.Name(), ".img") == len(name)-4 {
			images = append(images, CloudImage{
				Name:    info.Name(),
				Path:    hostpath,
				Size:    info.Size(),
				Created: info.ModTime(),
			})
		}
		return nil
	})

	return
}

// ListImages on premise
func (p *OnPrem) ListImages(ctx *Context) error {
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
		row = append(row, Bytes2Human(i.Size))
		row = append(row, Time2Human(i.Created))
		table.Append(row)
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
func (p *OnPrem) SyncImage(config *config.Config, target Provider, image string) error {
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
