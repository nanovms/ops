package lepton

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/olekukonko/tablewriter"
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
			if val, ok := s.Metadata["CreatedBy"]; ok && val == "ops" {
				ipv4 := ""

				// addresses may have a different structure in each cloud provider
				// vexx
				z := s.Addresses["public"]
				if z == nil {
					// ovh
					z = s.Addresses["Ext-Net"]
				}

				if z != nil {
					for _, v := range z.([]interface{}) {
						sz := v.(map[string]interface{})
						version := sz["version"].(float64)
						if version == 4 {
							ipv4 = sz["addr"].(string)
						}
					}
				}

				cinstance := CloudInstance{
					ID:      s.ID,
					Name:    s.Name,
					Status:  s.Status,
					Created: s.Created.Format("2006-01-02 15:04:05"),
				}

				if ipv4 != "" {
					cinstance.PublicIps = []string{ipv4}
				}

				cinstances = append(cinstances, cinstance)
			}
		}

		return true, nil
	})

	return cinstances, nil
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

	fmt.Printf("deploying imageID %s\n", imageID)

	flavorID, err := o.findFlavorByName(ctx.config.CloudConfig.Flavor)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Deploying flavorID %s\n", flavorID)

	instanceName := ctx.config.RunConfig.InstanceName

	var createOpts servers.CreateOptsBuilder
	createOpts = &servers.CreateOpts{
		Name:      instanceName,
		ImageRef:  imageID,
		FlavorRef: flavorID,
		AdminPass: "TODO",
		Metadata:  map[string]string{"CreatedBy": "ops"},
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

	fmt.Printf("Instance Created Successfully. ID ---> %s | Name ---> %s\n", server.ID, instanceName)

	if ctx.config.RunConfig.DomainName != "" {
		pollCount := 60
		for pollCount > 0 {
			fmt.Printf(".")
			time.Sleep(2 * time.Second)

			instance, err := o.GetInstanceByID(ctx, server.Name)
			if err != nil || len(instance.PublicIps) == 0 {
				pollCount--
				continue
			}

			if len(instance.PublicIps) != 0 {
				err := CreateDNSRecord(ctx.config, instance.PublicIps[0], o)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}

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
				fmt.Printf("Deleted instance with ID %s and name %s\n", instance.ID, instancename)
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
