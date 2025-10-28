//go:build azure || !onlyprovider

package azure

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

/*
	D - general purpose compute
	1 - number of cpus
	p - arm processor
	s - premium storage
	v5 - version
*/

func amendConfig(c *types.Config) {
	if strings.Contains(c.CloudConfig.Flavor, "p") {
		c.Uefi = true
		c.CloudConfig.ImageType = "gen2"
	}
}

// BuildImage to be upload on Azure
func (a *Azure) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	amendConfig(c)

	a.hyperVGen = armcompute.HyperVGenerationV1
	imageType := strings.ToLower(c.CloudConfig.ImageType)
	if imageType != "" {
		if imageType == "gen1" {
		} else if imageType == "gen2" {
			c.Uefi = true
			a.hyperVGen = armcompute.HyperVGenerationV2
		} else {
			return "", fmt.Errorf("invalid image type '%s'; available types: 'gen1', 'gen2'", c.CloudConfig.ImageType)
		}
	}
	err := lepton.BuildImage(*c)
	if err != nil {
		return "", err
	}

	return a.CustomizeImage(ctx)
}

func (a *Azure) getArchiveName(ctx *lepton.Context) string {
	return ctx.Config().CloudConfig.ImageName + ".tar.gz"
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (a *Azure) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}

// BuildImageWithPackage to upload on Azure
func (a *Azure) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {

	c := ctx.Config()
	amendConfig(c)

	a.hyperVGen = armcompute.HyperVGenerationV1
	imageType := strings.ToLower(c.CloudConfig.ImageType)
	if imageType != "" {
		if imageType == "gen1" {
		} else if imageType == "gen2" {
			c.Uefi = true
			a.hyperVGen = armcompute.HyperVGenerationV2
		} else {
			return "", fmt.Errorf("invalid image type '%s'; available types: 'gen1', 'gen2'", c.CloudConfig.ImageType)
		}
	}

	err := lepton.BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return a.CustomizeImage(ctx)
}

func (a *Azure) container() string {
	return "quickstart-nanos"
}

// CreateImage - Creates image on Azure using nanos images
func (a *Azure) CreateImage(ctx *lepton.Context, imagePath string) error {
	err := a.Storage.CopyToBucket(ctx.Config(), imagePath)
	if err != nil {
		return err
	}

	c := ctx.Config()
	imgName := c.CloudConfig.ImageName

	bucket, err := a.getBucketName()
	if err != nil {
		return err
	}

	container := a.container()
	disk := c.CloudConfig.ImageName + ".vhd"

	uri := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	location := a.getLocation(ctx.Config())

	ctx2 := context.Background()

	dName := c.CloudConfig.ImageName + ".vhd"

	images, err := a.GetImages(ctx, "")
	if err != nil {
		return err
	}

	for i := 0; i < len(images); i++ {
		if images[i].Name == imgName {
			fmt.Printf("image %s already exists - please delete this first", imgName)
			os.Exit(1)
		}
	}

	snapshot, err := a.createSnapshot(ctx2, location, dName, c.CloudConfig.ImageName)
	if err != nil {
		fmt.Println(err)
	}
	ctx.Logger().Debugf("snapshot: %+v", *snapshot.ID)

	gallery, err := a.createGallery(ctx2, location)
	if err != nil {
		fmt.Println(err)
	}
	ctx.Logger().Debugf("gallery:", *gallery.ID)

	flavor := c.CloudConfig.Flavor

	galleryImage, err := a.createGalleryImage(ctx2, location, c.CloudConfig.ImageName, flavor)
	if err != nil {
		fmt.Println(err)
	}
	ctx.Logger().Debugf("gallery image:", *galleryImage.ID)

	galleryImageVersion, err := a.createGalleryImageVersion(ctx2, location, c.CloudConfig.ImageName, uri)
	if err != nil {
		fmt.Println(err)
	}
	ctx.Logger().Debugf("gallery image version:", *galleryImageVersion.ID)

	return nil
}

// GetImages return all images for azure
func (a *Azure) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
	var cimages []lepton.CloudImage

	pager := a.clientFactory.NewGalleryImagesClient().NewListByGalleryPager(a.groupName, a.galleryName(), nil)
	for pager.More() {
		page, err := pager.NextPage(context.TODO())
		if err != nil {
			log.Fatalf("failed to advance page: %v", err)
		}

		for _, v := range page.Value {
			//			fmt.Printf("%+v", v)

			var t time.Time
			val, ok := (*v).Tags["CreatedAt"]
			if ok {
				t, err = time.Parse(time.RFC3339, *val)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				t = time.Now() // hack
			}

			ps := (*v).Properties.ProvisioningState
			status := string(armcompute.GalleryProvisioningState(*ps))
			cImage := lepton.CloudImage{
				Name:    *v.Name,
				Created: t,
				Status:  status,
			}

			cimages = append(cimages, cImage)

		}
	}

	return cimages, nil
}

// ListImages lists images on azure
func (a *Azure) ListImages(ctx *lepton.Context, filter string) error {

	cimages, err := a.GetImages(ctx, "")
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

	for _, image := range cimages {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, fmt.Sprintf("%v", image.Created))
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from Azure
func (a *Azure) DeleteImage(ctx *lepton.Context, imagename string) error {
	// snapshot
	// can/should we delete this on import?

	ctx2 := context.Background()

	// gallery image version
	givp, err := a.clientFactory.NewGalleryImageVersionsClient().BeginDelete(ctx2, a.groupName, a.galleryName(), imagename, "1.0.0", nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = givp.PollUntilDone(ctx2, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	ctx.Logger().Debug("deleted gallery image version")

	// race?

	// gallery image
	gip, err := a.clientFactory.NewGalleryImagesClient().BeginDelete(ctx2, a.groupName, a.galleryName(), imagename, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = gip.PollUntilDone(ctx2, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	ctx.Logger().Debug("deleted gallery image")

	// snapshot
	sp, err := a.clientFactory.NewSnapshotsClient().BeginDelete(ctx2, a.groupName, imagename, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	_, err = sp.PollUntilDone(ctx2, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	ctx.Logger().Debug("deleted snapshot")

	err = a.Storage.DeleteFromBucket(ctx.Config(), imagename+".vhd")
	if err != nil {
		return err
	}

	return nil
}

func (a *Azure) createSnapshot(ctx context.Context, location string, dName string, imageName string) (*armcompute.Snapshot, error) {
	snapshotsClient := a.clientFactory.NewSnapshotsClient()

	bucket, err := a.getBucketName()
	if err != nil {
		return nil, err
	}

	container := a.container()
	disk := dName

	uri := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	StorageAccountID := a.getStorageAccountID(bucket)

	snapshotName := imageName
	resourceGroupName := a.groupName
	pollerResp, err := snapshotsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		snapshotName,
		armcompute.Snapshot{
			Location: to.Ptr(location),
			Properties: &armcompute.SnapshotProperties{
				CreationData: &armcompute.CreationData{
					CreateOption:     to.Ptr(armcompute.DiskCreateOptionImport),
					SourceURI:        to.Ptr(uri),
					StorageAccountID: to.Ptr(StorageAccountID),
				},
				DiskSizeGB: to.Ptr(int32(1)), // had to set this for arm but not x86 ?
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.Snapshot, nil
}

func (a *Azure) createGallery(ctx context.Context, location string) (*armcompute.Gallery, error) {
	galleriesClient := a.clientFactory.NewGalleriesClient()

	pollerResp, err := galleriesClient.BeginCreateOrUpdate(
		ctx,
		a.groupName,
		a.galleryName(),
		armcompute.Gallery{
			Location: to.Ptr(location),
			Properties: &armcompute.GalleryProperties{
				Description: to.Ptr("nanos image gallery"),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.Gallery, nil
}

func (a *Azure) createGalleryImage(ctx context.Context, location string, imageName string, flavor string) (*armcompute.GalleryImage, error) {
	galleryImagesClient := a.clientFactory.NewGalleryImagesClient()

	galleryImageName := imageName
	resourceGroupName := a.groupName

	arch := armcompute.ArchitectureX64
	if strings.Contains(flavor, "p") {
		arch = armcompute.ArchitectureArm64
	}

	pollerResp, err := galleryImagesClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		a.galleryName(),
		galleryImageName,
		armcompute.GalleryImage{
			Location: to.Ptr(location),

			Properties: &armcompute.GalleryImageProperties{
				Architecture:     to.Ptr(arch),
				OSType:           to.Ptr(armcompute.OperatingSystemTypesLinux),
				OSState:          to.Ptr(armcompute.OperatingSystemStateTypesGeneralized),
				HyperVGeneration: to.Ptr(a.hyperVGen),
				Identifier: &armcompute.GalleryImageIdentifier{
					Offer:     to.Ptr("nanovms"),
					Publisher: to.Ptr("myOfferName"),
					SKU:       to.Ptr(imageName),
				},
			},
			Tags: getAzureDefaultTags(),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.GalleryImage, nil
}

func (a *Azure) galleryName() string {
	galleryName := os.Getenv("AZURE_GALLERY_NAME")
	if galleryName != "" {
		return galleryName
	}

	return "nanos_gallery"
}

func (a *Azure) getStorageAccountID(bucket string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s", a.subID, a.groupName, bucket)
}

func (a *Azure) createGalleryImageVersion(ctx context.Context, location string, imageName string, uri string) (*armcompute.GalleryImageVersion, error) {
	galleryImagesClient := a.clientFactory.NewGalleryImageVersionsClient()

	galleryImageName := imageName
	resourceGroupName := a.groupName

	bucket, err := a.getBucketName()
	if err != nil {
		return nil, err
	}

	StorageAccountID := a.getStorageAccountID(bucket)

	poller, err := galleryImagesClient.BeginCreateOrUpdate(ctx, resourceGroupName, a.galleryName(), galleryImageName, "1.0.0", armcompute.GalleryImageVersion{
		Location: to.Ptr(location),
		Properties: &armcompute.GalleryImageVersionProperties{
			PublishingProfile: &armcompute.GalleryImageVersionPublishingProfile{
				ReplicationMode: to.Ptr(armcompute.ReplicationModeShallow), // this makes it 'fast' like managed images but if you want to copy from region to region it should not be set.
			},
			SafetyProfile: &armcompute.GalleryImageVersionSafetyProfile{
				AllowDeletionOfReplicatedLocations: to.Ptr(false),
			},
			StorageProfile: &armcompute.GalleryImageVersionStorageProfile{
				OSDiskImage: &armcompute.GalleryOSDiskImage{
					HostCaching: to.Ptr(armcompute.HostCachingReadOnly),
					Source: &armcompute.GalleryDiskImageSource{
						StorageAccountID: to.Ptr(StorageAccountID),
						URI:              to.Ptr(uri),
					},
				},
			},
		},
	}, nil)

	if err != nil {
		return nil, err
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp.GalleryImageVersion, err
}

// SyncImage syncs image from provider to another provider
func (a *Azure) SyncImage(config *types.Config, target lepton.Provider, image string) error {
	log.Warn("not yet implemented")
	return nil
}

// ResizeImage is not supported on azure.
func (a *Azure) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}
