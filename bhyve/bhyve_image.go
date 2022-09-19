package bhyve

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// BuildImage to be upload on bhyve
func (p *Bhyve) BuildImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (p *Bhyve) CustomizeImage(ctx *lepton.Context) (string, error) {
	return "", nil
}

// BuildImageWithPackage to upload on bhyve
func (p *Bhyve) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	return "", nil
}

// CreateImage - Creates image on bhyve using nanos images
func (p *Bhyve) CreateImage(ctx *lepton.Context, imagePath string) error {
	return nil
}

// GetImages return all images on Bhyve
func (p *Bhyve) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	return nil, nil
}

// ListImages lists images on Bhyve
func (p *Bhyve) ListImages(ctx *lepton.Context) error {
	return nil
}

// DeleteImage deletes image from Bhyve
func (p *Bhyve) DeleteImage(ctx *lepton.Context, imagename string) error {
	return nil
}

// SyncImage syncs image from provider to another provider
func (p *Bhyve) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	return nil
}

// ResizeImage is not supported on bhyve
func (p *Bhyve) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}
