package lepton

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"

	"github.com/gophercloud/gophercloud/pagination"
)

func getOpenStackInstances(provider *gophercloud.ProviderClient, opts servers.ListOpts) ([]CloudInstance, error) {
	cinstances := []CloudInstance{}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println(err)
	}

	pager := servers.List(client, opts)

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			fmt.Println(err)
			return false, err
		}

		for _, s := range serverList {
			// fugly
			ipv4 := ""
			// For some instances IP is not assigned.
			z := s.Addresses["public"]
			if z != nil {
				for _, v := range z.([]interface{}) {
					sz := v.(map[string]interface{})
					version := sz["version"].(float64)
					if version == 4 {
						ipv4 = sz["addr"].(string)
					}
				}
			} else {
				ipv4 = "NA"
			}

			cinstance := CloudInstance{
				ID:        s.ID,
				Name:      s.Name,
				PublicIps: []string{ipv4},
				Status:    s.Status,
				Created:   s.Created.Format("2006-01-02 15:04:05"),
			}

			cinstances = append(cinstances, cinstance)
		}

		return true, nil
	})

	return cinstances, nil
}

// OpenStack provides access to the OpenStack API.
type OpenStack struct {
	Storage  *Datastores
	provider *gophercloud.ProviderClient
}

// ResizeImage is not supported on OpenStack.
func (o *OpenStack) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// BuildImage to be upload on OpenStack
func (o *OpenStack) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return o.customizeImage(ctx)
}

// BuildImageWithPackage to upload on OpenStack.
func (o *OpenStack) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return o.customizeImage(ctx)
}

// Initialize OpenStack related things
func (o *OpenStack) Initialize() error {

	opts, err := openstack.AuthOptionsFromEnv()

	if err != nil {
		fmt.Println(err)
	}

	o.provider, err = openstack.AuthenticatedClient(opts)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (o *OpenStack) findImage(name string) (id string, err error) {

	imageClient, err := openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println(err)
	}

	listOpts := images.ListOpts{
		Name: name,
	}

	allPages, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		panic(err)
	}

	// yolo
	// names are not unique so this is just (for now) grabbing first
	// result
	// FIXME
	if len(allImages) > 0 {
		return allImages[0].ID, nil
	}

	return "", errors.New("not found")
}

func (o *OpenStack) getImagesClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}

func (o *OpenStack) createImage(imagesClient *gophercloud.ServiceClient, imgName string) (*images.Image, error) {
	visibility := images.ImageVisibilityPrivate

	createOpts := images.CreateOpts{
		Name:            imgName,
		DiskFormat:      "raw",
		ContainerFormat: "bare",
		Visibility:      &visibility,
	}

	return images.Create(imagesClient, createOpts).Extract()
}

func (o *OpenStack) uploadImage(imagesClient *gophercloud.ServiceClient, imageID string, imagePath string) error {
	imageData, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imageData.Close()

	return imagedata.Upload(imagesClient, imageID, imageData).ExtractErr()
}

// CreateImage - Creates image on OpenStack using nanos images
func (o *OpenStack) CreateImage(ctx *Context) error {
	c := ctx.config
	imgName := c.CloudConfig.ImageName

	imgName = strings.ReplaceAll(imgName, "-image", "")

	fmt.Println("creating image:\t" + imgName)

	imagesClient, err := o.getImagesClient()
	if err != nil {
		fmt.Println(err)
	}

	image, err := o.createImage(imagesClient, imgName)
	if err != nil {
		fmt.Println(err)
	}

	imagePath := localImageDir + "/" + imgName
	err = o.uploadImage(imagesClient, image.ID, imagePath)
	if err != nil {
		return err
	}

	return nil
}

// GetImages return all images for openstack
func (o *OpenStack) GetImages(ctx *Context) ([]CloudImage, error) {
	var cimages []CloudImage

	imageClient, err := openstack.NewImageServiceV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		fmt.Println(err)
	}

	listOpts := images.ListOpts{}

	allPages, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		fmt.Println(err)
	}

	for _, image := range allImages {

		cimage := CloudImage{
			Name:    image.Name,
			Status:  string(image.Status),
			Created: time2Human(image.CreatedAt),
		}

		cimages = append(cimages, cimage)
	}

	return cimages, nil
}

// ListImages lists images on a datastore.
func (o *OpenStack) ListImages(ctx *Context) error {

	cimages, err := o.GetImages(ctx)
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
		row = append(row, image.Created)

		table.Append(row)
	}

	table.Render()

	return nil
}

func (o *OpenStack) deleteImage(imagesClient *gophercloud.ServiceClient, imageID string) error {
	return images.Delete(imagesClient, imageID).ExtractErr()
}

// DeleteImage deletes image from OpenStack
func (o *OpenStack) DeleteImage(ctx *Context, imagename string) error {
	imageID, err := o.findImage(imagename)
	if err != nil {
		fmt.Println(err)
		return err
	}

	imageClient, err := o.getImagesClient()
	if err != nil {
		fmt.Println(err)
	}

	err = images.Delete(imageClient, imageID).ExtractErr()
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// SyncImage syncs image from provider to another provider
func (o *OpenStack) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

func (o *OpenStack) findFlavorByName(name string) (id string, err error) {
	client, err := o.getComputeClient()
	if err != nil {
		fmt.Println(err)
	}

	listOpts := flavors.ListOpts{
		AccessType: flavors.PublicAccess,
	}

	allPages, err := flavors.ListDetail(client, listOpts).AllPages()
	if err != nil {
		panic(err)
	}

	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		panic(err)
	}

	if name == "" {
		// setting first flavor as default in case not provided
		return allFlavors[0].ID, nil
	}

	for _, flavor := range allFlavors {
		if flavor.Name == name {
			return flavor.ID, nil
		}
	}

	return "", errors.New("\nflavor " + name + " not found")
}

func (o *OpenStack) getComputeClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}

// CreateInstance - Creates instance on OpenStack.
func (o *OpenStack) CreateInstance(ctx *Context) error {
	client, err := o.getComputeClient()
	if err != nil {
		fmt.Println(err)
	}

	imageName := ctx.config.CloudConfig.ImageName

	imageID, err := o.findImage(imageName)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Printf("deploying imageID %s", imageID)

	flavorID, err := o.findFlavorByName(ctx.config.CloudConfig.Flavor)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("\nDeploying flavorID %s", flavorID)

	instanceName := imageName + "-" + strconv.FormatInt(time.Now().Unix(), 10)

	var createOpts servers.CreateOptsBuilder
	createOpts = &servers.CreateOpts{
		Name:      instanceName,
		ImageRef:  imageID,
		FlavorRef: flavorID,
		AdminPass: "TODO",
	}

	var volumeSize int
	if ctx.config.RunConfig.VolumeSizeInGb == 0 {
		volumeSize = 1
	} else {
		volumeSize = ctx.config.RunConfig.VolumeSizeInGb
	}

	createOpts = o.addBootFromVolumeParams(createOpts, imageID, volumeSize)
	server, err := servers.Create(client, createOpts).Extract()

	if err != nil {
		exitWithError(err.Error())
	}

	fmt.Printf("\nInstance Created Successfully. ID ---> %s | Name ---> %s\n", server.ID, instanceName)
	return nil
}

func (o *OpenStack) addBootFromVolumeParams(
	createOpts servers.CreateOptsBuilder,
	imageID string,
	rootDiskSizeGb int,
) *bootfromvolume.CreateOptsExt {
	blockDevice := bootfromvolume.BlockDevice{
		BootIndex:           0,
		DeleteOnTermination: true,
		DestinationType:     "volume",
		SourceType:          bootfromvolume.SourceType("image"),
		UUID:                imageID,
	}
	if rootDiskSizeGb > 0 {
		blockDevice.VolumeSize = rootDiskSizeGb
	}

	return &bootfromvolume.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		BlockDevice:       []bootfromvolume.BlockDevice{blockDevice},
	}
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (o *OpenStack) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	opts := servers.ListOpts{
		Name: id,
	}

	instances, err := getOpenStackInstances(o.provider, opts)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, ErrInstanceNotFound(id)
	}

	return &instances[0], nil
}

// GetInstances return all instances on OpenStack
func (o *OpenStack) GetInstances(ctx *Context) ([]CloudInstance, error) {
	return getOpenStackInstances(o.provider, servers.ListOpts{})
}

// ListInstances lists instances on OpenStack.
// It essentially does:
func (o *OpenStack) ListInstances(ctx *Context) error {
	cinstances, err := o.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "IP", "Status", "Created"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range cinstances {
		var row []string

		row = append(row, instance.ID)
		row = append(row, instance.Name)
		row = append(row, strings.Join(instance.PublicIps, ","))
		row = append(row, instance.Status)
		row = append(row, instance.Created)

		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from OpenStack
func (o *OpenStack) DeleteInstance(ctx *Context, instancename string) error {

	instances, err := o.GetInstances(ctx)

	client, err := openstack.NewComputeV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	if err != nil {
		return err
	}

	if len(instances) == 0 {
		exitWithError("No Instance available for deletion")
	}

	for _, instance := range instances {
		if instance.Name == instancename {
			result := servers.Delete(client, instance.ID).ExtractErr()

			if result == nil {
				fmt.Printf("Deleted instance with ID %s and name %s", instance.ID, instancename)
			} else {
				exitWithError(result.Error())
			}

		}
	}

	return nil
}

// StartInstance starts an instance in OpenStack.
func (o *OpenStack) StartInstance(ctx *Context, instancename string) error {
	client, err := o.getComputeClient()
	if err != nil {
		fmt.Println(err)
	}

	server, err := o.findInstance(instancename)
	if err != nil {
		fmt.Println(err)
	}

	err = startstop.Start(client, server.ID).ExtractErr()
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// StopInstance stops an instance from OpenStack
func (o *OpenStack) StopInstance(ctx *Context, instancename string) error {
	client, err := o.getComputeClient()
	if err != nil {
		fmt.Println(err)
	}

	server, err := o.findInstance(instancename)
	if err != nil {
		fmt.Println(err)
	}

	err = startstop.Stop(client, server.ID).ExtractErr()
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (o *OpenStack) findInstance(name string) (volume *servers.Server, err error) {
	var server *servers.Server

	client, err := o.getComputeClient()
	if err != nil {
		fmt.Println(err)
	}

	opts := servers.ListOpts{}

	pager := servers.List(client, opts)

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			fmt.Println(err)
			return false, err
		}

		for _, s := range serverList {
			if s.Name == name {
				server = &s
				return true, nil
			}
		}

		return true, nil
	})

	if server != nil {
		return server, nil
	}

	return nil, errors.New("could not find server")
}

// PrintInstanceLogs writes instance logs to console
func (o *OpenStack) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := o.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs.
func (o *OpenStack) GetInstanceLogs(ctx *Context, instancename string) (string, error) {

	client, err := o.getComputeClient()
	if err != nil {
		return "", err
	}

	server, err := o.findInstance(instancename)
	if err != nil {
		return "", err
	}

	outputOpts := &servers.ShowConsoleOutputOpts{
		Length: 100,
	}
	output, err := servers.ShowConsoleOutput(client, server.ID, outputOpts).Extract()
	if err != nil {
		return "", err
	}

	return output, nil
}

func (o *OpenStack) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	return imagePath, nil
}

// GetStorage returns storage interface for cloud provider
func (o *OpenStack) GetStorage() Storage {
	return o.Storage
}
