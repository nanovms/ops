package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// GetVM gets the specified VM info
func (a *Azure) GetVM(ctx context.Context, vmName string) (vm compute.VirtualMachine, err error) {
	vmClient := a.getVMClient()

	vm, err = vmClient.Get(ctx, a.groupName, vmName, compute.InstanceView)
	return
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

	location := a.getLocation(ctx.Config())

	vmName := ctx.Config().RunConfig.InstanceName
	ctx.Logger().Log("spinning up:\t%s", vmName)

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
		ctx.Logger().Info("creating virtual network with id %s", vmName)
		vnet, err = a.CreateVirtualNetwork(context.TODO(), location, vmName)
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
		ctx.Logger().Info("creating network security group with id %s", vmName)
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
		ctx.Logger().Info("creating subnet with id %s", vmName)
		subnet, err = a.CreateSubnetWithNetworkSecurityGroup(context.TODO(), *vnet.Name, vmName, "10.0.0.0/24", *nsg.Name)
		if err != nil {
			ctx.Logger().Error(err)
			return errors.New("error creating subnet")
		}
	}

	// create ip
	ctx.Logger().Info("creating public ip with id %s", vmName)
	ip, err := a.CreatePublicIP(context.TODO(), location, vmName)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("error creating public ip")
	}

	// create nic
	// pass vnet, subnet, ip, nicname
	ctx.Logger().Info("creating network interface controller with id %s", vmName)
	nic, err := a.CreateNIC(context.TODO(), location, *vnet.Name, *subnet.Name, *nsg.Name, *ip.Name, vmName)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("error creating network interface controller")
	}

	var sshKeyData string
	sshKeyData = fakepubkey
	nctx := context.TODO()

	ctx.Logger().Log("creating the vm - this can take a few minutes")

	vmClient := a.getVMClient()

	var flavor compute.VirtualMachineSizeTypes
	flavor = compute.VirtualMachineSizeTypes(ctx.Config().CloudConfig.Flavor)
	if flavor == "" {
		flavor = compute.VirtualMachineSizeTypesStandardB1s
	}

	tags := getAzureDefaultTags()
	tags["image"] = &ctx.Config().CloudConfig.ImageName

	future, err := vmClient.CreateOrUpdate(
		nctx,
		a.groupName,
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(location),
			Tags:     tags,
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: flavor,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						ID: to.StringPtr("/subscriptions/" + a.subID + "/resourceGroups/" + a.groupName + "/providers/Microsoft.Compute/images/" + ctx.Config().CloudConfig.ImageName),
					},
				},
				DiagnosticsProfile: &compute.DiagnosticsProfile{
					BootDiagnostics: &compute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr("https://" + bucket + ".blob.core.windows.net/"),
					},
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmName),
					AdminUsername: to.StringPtr(username),
					AdminPassword: to.StringPtr(password),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path: to.StringPtr(
										fmt.Sprintf("/home/%s/.ssh/authorized_keys",
											username)),
									KeyData: to.StringPtr(sshKeyData),
								},
							},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						{
							ID: nic.ID,
							NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
								Primary: to.BoolPtr(true),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatal("cannot create vm: %v\n", err.Error())
	}

	err = future.WaitForCompletionRef(nctx, vmClient.Client)
	if err != nil {
		log.Fatal("cannot get the vm create or update future response: %v\n", err.Error())
	}

	_, err = future.Result(*vmClient)
	if err != nil {
		log.Error(err)
	}

	if ctx.Config().CloudConfig.DomainName != "" {
		err = lepton.CreateDNSRecord(ctx.Config(), *ip.IPAddress, a)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (a *Azure) GetInstanceByID(ctx *lepton.Context, id string) (vm *lepton.CloudInstance, err error) {
	vmClient := a.getVMClient()

	nicClient := a.getNicClient()
	ipClient := a.getIPClient()

	result, err := vmClient.Get(context.TODO(), a.groupName, id, compute.InstanceView)
	if err != nil {
		return nil, err
	}

	return a.convertToCloudInstance(&result, nicClient, ipClient)
}

// GetInstances return all instances on Azure
func (a *Azure) GetInstances(ctx *lepton.Context) (cinstances []lepton.CloudInstance, err error) {
	vmClient := a.getVMClient()
	nicClient := a.getNicClient()
	ipClient := a.getIPClient()

	vmlist, err := vmClient.List(context.TODO(), a.groupName)
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

	ctx.Logger().Info("Getting vm with ID %s...", instancename)
	vm, err := a.GetVM(context.TODO(), instancename)
	if err != nil {
		return err
	}

	ctx.Logger().Info("Deleting vm with ID %s...", instancename)
	future, err := vmClient.Delete(context.TODO(), a.groupName, instancename)
	if err != nil {
		ctx.Logger().Error(err)
		return errors.New("Unable to delete instance")
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

			for _, ipConfiguration := range *nic.IPConfigurations {
				err := a.DeleteIP(ctx, &ipConfiguration)
				if err != nil {
					ctx.Logger().Warn(err.Error())
				}
			}

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
	fmt.Printf(l)
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

	vm, err := vmClient.Get(context.TODO(), a.groupName, vmName, compute.InstanceView)
	if err != nil {
		log.Fatal(err.Error())
	}

	// this is unique per vm || per boot?
	vmid := to.String(vm.VMID)

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

	get, err := blobURL.Download(context.TODO(), 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return "", err
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(azblob.RetryReaderOptions{})
	downloadedData.ReadFrom(reader)
	reader.Close()

	return downloadedData.String(), nil
}
