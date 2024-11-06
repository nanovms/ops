//go:build azure || !onlyprovider

package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"

	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-07-02/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2022-07-01/network"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// GetVM gets the specified VM info
func (a *Azure) GetVM(ctx context.Context, vmName string) (vm compute.VirtualMachine, err error) {
	vmClient := a.getVMClient()

	vm, err = vmClient.Get(ctx, a.groupName, vmName, compute.InstanceViewTypesInstanceView)
	return
}

// RebootInstance reboots the instance.
// prob use the below?
func (a *Azure) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// RestartVM restarts the selected VM
func (a *Azure) RestartVM(ctx context.Context, vmName string) (osr autorest.Response, err error) {
	vmClient := a.getVMClient()

	future, err := vmClient.Restart(ctx, a.groupName, vmName)
	if err != nil {
		return osr, fmt.Errorf("cannot restart vm: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return osr, fmt.Errorf("cannot get the vm restart future response: %v", err)
	}

	return future.Result(*vmClient)
}

// CreateInstance - Creates instance on azure Platform
func (a *Azure) CreateInstance(ctx *lepton.Context) error {
	username := "fake"
	password := "fake"

	c := ctx.Config()

	bucket, err := a.getBucketName()
	if err != nil {
		return err
	}

	location, az := stripAndExtractAvailibilityZone(a.getLocation(ctx.Config()))

	vmName := ctx.Config().RunConfig.InstanceName
	ctx.Logger().Logf("spinning up:\t%s", vmName)

	// create virtual network
	var vnet *network.VirtualNetwork
	configVPC := ctx.Config().CloudConfig.VPC
	if configVPC != "" {
		vnet, err = a.GetVPC(configVPC)
		if err != nil {
			ctx.Logger().Error(err)
			return fmt.Errorf("error getting virtual network with id %s", configVPC)
		}
	} else {
		ctx.Logger().Infof("creating virtual network with id %s", vmName)
		vnet, err = a.CreateVirtualNetwork(context.TODO(), location, vmName, c)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error creating virtual network")
		}
	}

	// create nsg
	var nsg *network.SecurityGroup
	configSecurityGroup := ctx.Config().CloudConfig.SecurityGroup
	if configSecurityGroup != "" {
		nsg, err = a.GetNetworkSecurityGroup(context.TODO(), configSecurityGroup)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error getting security group")
		}
	} else {
		ctx.Logger().Infof("creating network security group with id %s", vmName)
		nsg, err = a.CreateNetworkSecurityGroup(context.TODO(), location, vmName, c)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error creating network security group")
		}
	}

	// create subnet
	var subnet *network.Subnet
	configSubnet := ctx.Config().CloudConfig.Subnet
	if configSubnet != "" {
		subnet, err = a.GetVirtualNetworkSubnet(context.TODO(), *vnet.Name, configSubnet)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error getting subnet")
		}
	} else {
		ctx.Logger().Infof("creating subnet with id %s", vmName)
		subnet, err = a.CreateSubnetWithNetworkSecurityGroup(context.TODO(), *vnet.Name, vmName, "10.0.0.0/24", *nsg.Name, c)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error creating subnet")
		}
	}

	// create ip
	ctx.Logger().Infof("creating public ip with id %s", vmName)
	ip, err := a.CreatePublicIP(context.TODO(), location, vmName, false)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("error creating public ip")
	}

	var ipv6Name network.PublicIPAddress
	v6name := ""

	if ctx.Config().CloudConfig.EnableIPv6 {
		ctx.Logger().Infof("creating public ip with id %s", vmName)
		ipv6Name, err = a.CreatePublicIP(context.TODO(), location, vmName, true)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error creating public ip")
		}

		if ipv6Name.Name != nil {
			v6name = *ipv6Name.Name
		}
	}

	// create nic
	// pass vnet, subnet, ip, nicname
	enableIPForwarding := c.RunConfig.CanIPForward
	ctx.Logger().Infof("creating network interface controller with id %s", vmName)

	nic, err := a.CreateNIC(context.TODO(), location, *vnet.Name, *subnet.Name, *nsg.Name, *ip.Name, v6name, vmName, enableIPForwarding, c)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("error creating network interface controller")
	}

	sshKeyData := fakepubkey

	ctx.Logger().Log("creating the vm - this can take a few minutes")

	//	vmClient := a.getVMClient()

	var flavor armcompute.VirtualMachineSizeTypes
	flavor = armcompute.VirtualMachineSizeTypes(ctx.Config().CloudConfig.Flavor)
	if flavor == "" {
		flavor = armcompute.VirtualMachineSizeTypesStandardB1S
	}

	computeClientFactory, err := armcompute.NewClientFactory(a.subID, a.cred, nil)
	if err != nil {
		log.Fatal(err)
	}

	virtualMachinesClient := computeClientFactory.NewVirtualMachinesClient()

	tags := getAzureDefaultTags()
	tags["image"] = &ctx.Config().CloudConfig.ImageName

	for _, tag := range ctx.Config().CloudConfig.Tags {
		tags[tag.Key] = to.Ptr(tag.Value)
	}

	var zone []*string = nil
	if az != "" {
		zone = append(zone, to.Ptr(az))
	}

	parameters := armcompute.VirtualMachine{
		Location: to.Ptr(location),
		Zones:    zone,
		Tags:     tags,
		Identity: &armcompute.VirtualMachineIdentity{
			Type: to.Ptr(armcompute.ResourceIdentityTypeNone),
		},
		Properties: &armcompute.VirtualMachineProperties{
			DiagnosticsProfile: &armcompute.DiagnosticsProfile{
				BootDiagnostics: &armcompute.BootDiagnostics{
					Enabled:    to.Ptr(true),
					StorageURI: to.Ptr("https://" + bucket + ".blob.core.windows.net/"),
				},
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					ID: to.Ptr("/subscriptions/" + a.subID + "/resourceGroups/" + a.groupName + "/providers/Microsoft.Compute/galleries/" + a.galleryName() + "/images/" + ctx.Config().CloudConfig.ImageName),
				},
				OSDisk: &armcompute.OSDisk{
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
					DeleteOption: to.Ptr(armcompute.DiskDeleteOptionTypesDelete),
					Caching:      to.Ptr(armcompute.CachingTypesReadWrite),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardLRS),
					},
					DiskSizeGB: to.Ptr[int32](1),
				},
			},
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: &flavor,
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  to.Ptr(vmName),
				AdminUsername: to.Ptr(username),
				AdminPassword: to.Ptr(password),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					SSH: &armcompute.SSHConfiguration{
						PublicKeys: []*armcompute.SSHPublicKey{
							{
								Path: to.Ptr(
									fmt.Sprintf("/home/%s/.ssh/authorized_keys",
										username)),
								KeyData: to.Ptr(sshKeyData),
							},
						},
					},
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: nic.ID,
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: to.Ptr(true),
						},
					},
				},
			},
		},
	}

	ctx2 := context.Background()

	pollerResponse, err := virtualMachinesClient.BeginCreateOrUpdate(ctx2, a.groupName, vmName, parameters, nil)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	_, err = pollerResponse.PollUntilDone(ctx2, nil)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	if ctx.Config().CloudConfig.DomainName != "" {
		err = lepton.CreateDNSRecord(ctx.Config(), *ip.IPAddress, a)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (a *Azure) GetInstanceByName(ctx *lepton.Context, name string) (vm *lepton.CloudInstance, err error) {
	vmClient := a.getVMClient()

	nicClient := a.getNicClient()
	ipClient := a.getIPClient()

	result, err := vmClient.Get(context.TODO(), a.groupName, name, compute.InstanceViewTypesInstanceView)
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			return nil, lepton.ErrInstanceNotFound(name)
		}
		return nil, err
	}

	return a.convertToCloudInstance(&result, nicClient, ipClient)
}

// GetInstances return all instances on Azure
func (a *Azure) GetInstances(ctx *lepton.Context) (cinstances []lepton.CloudInstance, err error) {
	vmClient := a.getVMClient()
	nicClient := a.getNicClient()
	ipClient := a.getIPClient()

	vmlist, err := vmClient.List(context.TODO(), a.groupName, "")
	if err != nil {
		return
	}

	instances := vmlist.Values()

	for _, instance := range instances {
		if hasAzureOpsTags(instance.Tags) {
			cinstance, err := a.convertToCloudInstance(&instance, nicClient, ipClient)
			if err != nil {
				return nil, err
			}

			if _, ok := instance.Tags["image"]; ok {
				cinstance.Image = *instance.Tags["image"]
			}

			cinstances = append(cinstances, *cinstance)
		}
	}

	return
}

func (a *Azure) convertToCloudInstance(instance *compute.VirtualMachine, nicClient *network.InterfacesClient, ipClient *network.PublicIPAddressesClient) (*lepton.CloudInstance, error) {
	cinstance := lepton.CloudInstance{
		Name: *instance.Name,
	}
	privateIP := ""
	publicIP := ""

	if instance.VirtualMachineProperties != nil {
		nifs := *((*(*instance.VirtualMachineProperties).NetworkProfile).NetworkInterfaces)

		for i := 0; i < len(nifs); i++ {
			nic, err := nicClient.Get(context.TODO(), a.groupName, cinstance.Name, "")
			if err != nil {
				return nil, err
			}

			if nic.InterfacePropertiesFormat != nil {
				ipconfig := *(*nic.InterfacePropertiesFormat).IPConfigurations
				for x := 0; x < len(ipconfig); x++ {
					format := *ipconfig[x].InterfaceIPConfigurationPropertiesFormat
					privateIP = *format.PrivateIPAddress
				}
			}
		}
	}

	pubip, err := ipClient.Get(context.TODO(), a.groupName, cinstance.Name, "")
	if err != nil {
		log.Error(err)
	}
	publicIP = *(*pubip.PublicIPAddressPropertiesFormat).IPAddress

	cinstance.PrivateIps = []string{privateIP}
	cinstance.PublicIps = []string{publicIP}

	return &cinstance, nil
}

// ListInstances lists instances on Azure
func (a *Azure) ListInstances(ctx *lepton.Context) error {
	cinstances, err := a.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips", "Image"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range cinstances {
		var rows []string

		rows = append(rows, instance.Name)
		rows = append(rows, "")
		rows = append(rows, "")
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))
		rows = append(rows, instance.Image)
		table.Append(rows)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from Azure
func (a *Azure) DeleteInstance(ctx *lepton.Context, instancename string) error {

	vmClient := a.getVMClient()

	ctx.Logger().Infof("Getting vm with ID %s...", instancename)
	vm, err := a.GetVM(context.TODO(), instancename)
	if err != nil {
		return err
	}

	ctx.Logger().Infof("Deleting vm with ID %s...", instancename)
	future, err := vmClient.Delete(context.TODO(), a.groupName, instancename, nil)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("unable to delete instance")
	}

	err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("error waiting for vm deletion")
	}

	ctx.Logger().Log("Instance deleted")
	ctx.Logger().Log("Deleting resources related with instance")

	nicClient := a.getNicClient()

	for _, nicReference := range *vm.NetworkProfile.NetworkInterfaces {
		nicID := getAzureResourceNameFromID(*nicReference.ID)

		nic, err := nicClient.Get(context.TODO(), a.groupName, nicID, "")
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("failed getting nic")
		}

		if hasAzureOpsTags(nic.Tags) {
			err = a.DeleteNIC(ctx, &nic)
			if err != nil {
				ctx.Logger().Warn(err.Error())
			}

			subnet := ""
			for _, ipConfiguration := range *nic.IPConfigurations {
				subnet = *ipConfiguration.Subnet.ID
				err := a.DeleteIP(ctx, &ipConfiguration)
				if err != nil {
					ctx.Logger().Warn(err.Error())
				}
			}

			a.DeleteSubnetwork(ctx, subnet)

			if nic.NetworkSecurityGroup != nil {
				err = a.DeleteNetworkSecurityGroup(ctx, *nic.NetworkSecurityGroup.ID)
				if err != nil {
					ctx.Logger().Warn(err.Error())
				}
			}
		}
	}

	ctx.Logger().Log("Instance deletion completed")
	return nil
}

// StartInstance starts an instance in Azure
func (a *Azure) StartInstance(ctx *lepton.Context, instancename string) error {

	vmClient := a.getVMClient()

	fmt.Printf("Starting instance %s", instancename)
	_, err := vmClient.Start(context.TODO(), a.groupName, instancename)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("failed starting virtual machine")
	}

	return nil
}

// StopInstance deletes instance from Azure
func (a *Azure) StopInstance(ctx *lepton.Context, instancename string) error {

	vmClient := a.getVMClient()
	// skipShutdown parameter is optional, we are taking its default
	// value here
	fmt.Printf("Stopping instance %s", instancename)
	_, err := vmClient.PowerOff(context.TODO(), a.groupName, instancename, nil)
	if err != nil {
		fmt.Printf("cannot power off vm: %v\n", err.Error())
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (a *Azure) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := a.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Println(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (a *Azure) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	// this is basically 2 calls
	// 1) grab the log location
	// 2) grab it from storage

	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		log.Warn("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		fmt.Printf("Invalid credentials with error: %s", err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/", accountName))

	containerURL := azblob.NewContainerURL(*URL, p)

	vmName := instancename

	vmClient := a.getVMClient()

	vm, err := vmClient.Get(context.TODO(), a.groupName, vmName, compute.InstanceViewTypesInstanceView)
	if err != nil {
		log.Fatal(err)
	}

	// this is unique per vm || per boot?
	vmid := *vm.VMID

	// this has a unique expected format apparently
	// the first part of the name in the uri is capped at 10 chars but
	// not the 2nd part?
	firstName := strings.ReplaceAll(vmName, "-", "")
	if len(firstName) > 10 {
		firstName = firstName[0:9]
	}

	fname := "bootdiagnostics" + "-" + firstName + "-" + vmid + "/" + vmName + "." + vmid +
		".serialconsole.log"

	blobURL := containerURL.NewBlockBlobURL(fname)

	get, err := blobURL.Download(context.TODO(), 0, 0, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return "", err
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(azblob.RetryReaderOptions{})
	downloadedData.ReadFrom(reader)
	reader.Close()

	return downloadedData.String(), nil
}

// InstanceStats show metrics for instances on azure.
func (a *Azure) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not avilable")
}
