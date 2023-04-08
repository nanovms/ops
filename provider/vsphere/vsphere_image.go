//go:build vsphere || !onlyprovider

package vsphere

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"
	vmwareTypes "github.com/vmware/govmomi/vim25/types"
)

// ResizeImage is not supported on VSphere.
func (v *Vsphere) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

// BuildImage to be upload on VSphere
func (v *Vsphere) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return v.CustomizeImage(ctx)
}

// BuildImageWithPackage to upload on Vsphere.
func (v *Vsphere) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return v.CustomizeImage(ctx)
}

func (v *Vsphere) createImage(key string, bucket string, region string) {
	log.Warn("un-implemented")
}

// CreateImage - Creates image on vsphere using nanos images
// This merely uploads the flat and base image to the datastore and then
// creates a copy of the image to perform the vmfs translation (import
// does not do this by default). This sidesteps the vmfkstools
// transformation.
func (v *Vsphere) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := v.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	vmdkBase := strings.ReplaceAll(ctx.Config().CloudConfig.ImageName, "-image", "")

	flat := vmdkBase + "-flat.vmdk"
	base := vmdkBase + ".vmdk"

	flatPath := "/tmp/" + flat
	imgPath := "/tmp/" + base

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		log.Error(err)
		return err
	}

	p := soap.DefaultUpload
	ds.UploadFile(context.TODO(), flatPath, vmdkBase+"/"+flat, &p)
	ds.UploadFile(context.TODO(), imgPath, vmdkBase+"/"+base, &p)

	dc, err := f.DatacenterOrDefault(context.TODO(), v.datacenter)
	if err != nil {
		log.Error(err)
		return err
	}

	m := ds.NewFileManager(dc, true)

	m.Copy(context.TODO(), vmdkBase+"/"+base, vmdkBase+"/"+vmdkBase+"2.vmdk")

	return nil
}

// GetImages return all images for vsphere
func (v *Vsphere) GetImages(ctx *lepton.Context) ([]lepton.CloudImage, error) {
	var cimages []lepton.CloudImage

	f := find.NewFinder(v.client, true)
	ds, err := f.DatastoreOrDefault(context.TODO(), v.datastore)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	b, err := ds.Browser(context.TODO())
	if err != nil {
		return nil, err
	}

	spec := vmwareTypes.HostDatastoreBrowserSearchSpec{
		MatchPattern: []string{"*"},
	}

	search := b.SearchDatastore

	task, err := search(context.TODO(), ds.Path(""), &spec)
	if err != nil {
		log.Error(err)
	}

	info, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Error(err)
	}

	switch r := info.Result.(type) {
	case vmwareTypes.HostDatastoreBrowserSearchResults:
		res := []vmwareTypes.HostDatastoreBrowserSearchResults{r}
		for i := 0; i < len(res); i++ {
			for _, f := range res[i].File {
				if f.GetFileInfo().Path[0] == '.' {
					continue
				}
				cimages = append(cimages, lepton.CloudImage{
					Name: f.GetFileInfo().Path,
				})
			}
		}
	case vmwareTypes.ArrayOfHostDatastoreBrowserSearchResults:
		log.Warn("un-implemented")
	}

	return cimages, nil
}

// ListImages lists images on a datastore.
// This is incredibly naive at the moment and probably worth putting
// under a root folder.
// essentially does the equivalent of 'govc datastore.ls'
func (v *Vsphere) ListImages(ctx *lepton.Context) error {
	images, err := v.GetImages(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range images {
		var row []string
		row = append(row, image.Name)
		row = append(row, "")
		row = append(row, "")
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from VSphere
func (v *Vsphere) DeleteImage(ctx *lepton.Context, imagename string) error {
	log.Warn("un-implemented")
	return nil
}

// SyncImage syncs image from provider to another provider
func (v *Vsphere) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (v *Vsphere) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}
