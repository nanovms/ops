package upcloud

import (
	"errors"
	"math"
	"os"

	"github.com/nanovms/ops/types"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

// BuildImage creates local image
func (p *Provider) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return c.RunConfig.Imagename, nil
}

// BuildImageWithPackage creates local image using package image
func (p *Provider) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}

	return ctx.Config().RunConfig.Imagename, nil
}

// CreateImage creates a storage object and upload image
func (p *Provider) CreateImage(ctx *lepton.Context, imagePath string) error {

	storageDetails, err := p.createStorage(ctx, ctx.Config().CloudConfig.ImageName, imagePath)
	if err != nil {
		return err
	}

	ctx.Logger().Info("creating custom image")
	templatizeReq := &request.TemplatizeStorageRequest{
		UUID:  storageDetails.UUID,
		Title: ctx.Config().CloudConfig.ImageName,
	}

	templateDetails, err := p.upcloud.TemplatizeStorage(templatizeReq)
	if err != nil {
		return err
	}

	ctx.Logger().Debug("%+v", templateDetails)

	err = p.waitForStorageState(storageDetails.UUID, "online")
	if err != nil {
		return err
	}

	err = p.deleteStorage(storageDetails.UUID)
	if err != nil {
		return err
	}

	return nil
}

// ListImages prints upcloud images in table format
func (p *Provider) ListImages(ctx *lepton.Context) error {
	images, err := p.GetImages(ctx)
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
func (p *Provider) GetImages(ctx *lepton.Context) (images []lepton.CloudImage, err error) {
	images = []lepton.CloudImage{}

	listTemplatesReq := &request.GetStoragesRequest{
		Type:   "template",
		Access: "private",
	}

	templates, err := p.upcloud.GetStorages(listTemplatesReq)
	if err != nil {
		return
	}

	ctx.Logger().Debug("%+v", templates)

	for _, s := range templates.Storages {
		images = append(images, *p.parseStorageToCloudImage(&s))
	}

	return
}

func (p *Provider) parseStorageToCloudImage(storage *upcloud.Storage) *lepton.CloudImage {
	return &lepton.CloudImage{
		ID:      storage.UUID,
		Name:    storage.Title,
		Status:  storage.State,
		Size:    int64(float64(storage.Size) * math.Pow(10, 9)),
		Created: storage.Created,
	}
}

// DeleteImage removes upcloud image
func (p *Provider) DeleteImage(ctx *lepton.Context, imagename string) (err error) {
	image, err := p.getImageByName(ctx, imagename)
	if err != nil {
		return
	}

	err = p.deleteStorage(image.ID)

	return
}

func (p *Provider) getImageByName(ctx *lepton.Context, imageName string) (image *lepton.CloudImage, err error) {
	images, err := p.GetImages(ctx)
	if err != nil {
		return
	}

	for _, i := range images {
		if i.Name == imageName {
			image = &i
			return
		}
	}

	err = errors.New("image not found")

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
