package lepton

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/olekukonko/tablewriter"
)

// most of this is ripped from the samples repo:
// https://github.com/Azure-Samples/azure-sdk-for-go-samples/blob/master/compute/vm.go
// the azure sdk is fairly round-a-bout and could use some heavy
// refactoring
const (
	userAgent  = "ops"
	fakepubkey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7laRyN4B3YZmVrDEZLZoIuUA72pQ0DpGuZBZWykCofIfCPrFZAJgFvonKGgKJl6FGKIunkZL9Us/mV4ZPkZhBlE7uX83AAf5i9Q8FmKpotzmaxN10/1mcnEE7pFvLoSkwqrQSkrrgSm8zaJ3g91giXSbtqvSIj/vk2f05stYmLfhAwNo3Oh27ugCakCoVeuCrZkvHMaJgcYrIGCuFo6q0Pfk9rsZyriIqEa9AtiUOtViInVYdby7y71wcbl0AbbCZsTSqnSoVxm2tRkOsXV6+8X4SnwcmZbao3H+zfO1GBhQOLxJ4NQbzAa8IJh810rYARNLptgmsd4cYXVOSosTX azureuser"
)

var (
	environment   *azure.Environment
	armAuthorizer autorest.Authorizer
	cloudName     = "AzurePublicCloud"
)

// Azure contains all operations for Azure
type Azure struct {
	Storage         *AzureStorage
	subID           string
	clientID        string
	tenantID        string
	clientSecret    string
	locationDefault string
	groupName       string
	storageAccount  string
}

// Environment returns an `azure.Environment{...}` for the current
// cloud.
func (a *Azure) Environment() *azure.Environment {
	if environment != nil {
		return environment
	}
	env, err := azure.EnvironmentFromName(cloudName)
	if err != nil {
		// TODO: move to initialization of var
		panic(fmt.Sprintf(
			"invalid cloud name '%s' specified, cannot continue\n", cloudName))
	}
	environment = &env
	return environment
}

func (a *Azure) getAuthorizerForResource(resource string) (autorest.Authorizer, error) {
	var authr autorest.Authorizer
	var err error

	oauthConfig, err := adal.NewOAuthConfig(
		a.Environment().ActiveDirectoryEndpoint, a.tenantID)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(
		*oauthConfig, a.clientID, a.clientSecret, resource)
	if err != nil {
		return nil, err
	}
	authr = autorest.NewBearerAuthorizer(token)

	return authr, err
}

// GetResourceManagementAuthorizer returns an autorest authorizer.
func (a *Azure) GetResourceManagementAuthorizer() (autorest.Authorizer, error) {
	if armAuthorizer != nil {
		return armAuthorizer, nil
	}

	var authr autorest.Authorizer
	var err error

	authr, err = a.getAuthorizerForResource(a.Environment().ResourceManagerEndpoint)

	if err == nil {
		// cache
		armAuthorizer = authr
	} else {
		// clear cache
		armAuthorizer = nil
	}
	return armAuthorizer, err
}

func (a *Azure) getImagesClient() compute.ImagesClient {
	vmClient := compute.NewImagesClientWithBaseURI(compute.DefaultBaseURI, a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	vmClient.Authorizer = authr
	vmClient.AddToUserAgent(userAgent)
	return vmClient
}

func (a *Azure) getVMClient() compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	vmClient.Authorizer = authr
	vmClient.AddToUserAgent(userAgent)
	return vmClient
}

func (a *Azure) getVMExtensionsClient() compute.VirtualMachineExtensionsClient {
	extClient := compute.NewVirtualMachineExtensionsClient(a.subID)
	authr, _ := a.GetResourceManagementAuthorizer()
	extClient.Authorizer = authr
	extClient.AddToUserAgent(userAgent)
	return extClient
}

func (a *Azure) getLocation(config *Config) string {
	c := config
	location := c.CloudConfig.Zone
	if location == "" {
		location = a.locationDefault
	}
	if location == "" {
		fmt.Println("Error: a location must be set via either the Zone attribute in CloudConfig or the AZURE_LOCATION_DEFAULT environment variable.")
		os.Exit(1)
	}
	return location
}

// GetVM gets the specified VM info
func (a *Azure) GetVM(ctx context.Context, vmName string) (compute.VirtualMachine, error) {
	vmClient := a.getVMClient()
	return vmClient.Get(ctx, a.groupName, vmName, compute.InstanceView)
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

	return future.Result(vmClient)
}

func (a *Azure) getArchiveName(ctx *Context) string {
	return ctx.config.CloudConfig.ImageName + ".tar.gz"
}

func (a *Azure) customizeImage(ctx *Context) (string, error) {
	imagePath := ctx.config.RunConfig.Imagename
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")

	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return "", fmt.Errorf("failed to unlink: %+v", err)
		}
	}

	err := os.Link(imagePath, symlink)
	if err != nil {
		return "", err
	}

	archPath := filepath.Join(filepath.Dir(imagePath), a.getArchiveName(ctx))
	files := []string{symlink}

	err = createArchive(archPath, files)
	if err != nil {
		return "", err
	}
	return archPath, nil
}

// BuildImage to be upload on Azure
func (a *Azure) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return "", err
	}

	return a.customizeImage(ctx)
}

// BuildImageWithPackage to upload on Azure
func (a *Azure) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return a.customizeImage(ctx)
}

// Initialize Azure related things
func (a *Azure) Initialize() error {
	a.Storage = &AzureStorage{}

	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subID != "" {
		a.subID = subID
	}

	locationDefault := os.Getenv("AZURE_LOCATION_DEFAULT")
	if locationDefault != "" {
		a.locationDefault = locationDefault
	}

	tenantID := os.Getenv("AZURE_TENANT_ID")
	if tenantID != "" {
		a.tenantID = tenantID
	}

	clientID := os.Getenv("AZURE_CLIENT_ID")
	if clientID != "" {
		a.clientID = clientID
	}

	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if clientSecret != "" {
		a.clientSecret = clientSecret
	}

	groupName := os.Getenv("AZURE_BASE_GROUP_NAME")
	if groupName != "" {
		a.groupName = groupName
	}

	storageAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if storageAccount != "" {
		a.storageAccount = storageAccount
	}

	return nil
}

// CreateImage - Creates image on Azure using nanos images
func (a *Azure) CreateImage(ctx *Context) error {
	imagesClient := a.getImagesClient()

	c := ctx.config
	imgName := c.CloudConfig.ImageName

	bucket := c.CloudConfig.BucketName

	region := a.getLocation(ctx.config)
	container := "quickstart-nanos"
	disk := c.CloudConfig.ImageName + ".vhd"

	uri := "https://" + bucket + ".blob.core.windows.net/" + container + "/" + disk

	imageParams := compute.Image{
		Location: to.StringPtr(region),
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					OsType:  compute.Linux,
					BlobURI: to.StringPtr(uri),
					OsState: compute.Generalized,
				},
			},
			HyperVGeneration: compute.HyperVGenerationTypesV1,
		},
	}

	res, err := imagesClient.CreateOrUpdate(context.TODO(), a.groupName, imgName, imageParams)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v", res)

	return nil
}

// GetImages return all images for azure
func (a *Azure) GetImages(ctx *Context) ([]CloudImage, error) {
	var cimages []CloudImage

	imagesClient := a.getImagesClient()

	images, err := imagesClient.List(context.TODO())
	if err != nil {
		return nil, err
	}

	imgs := images.Values()

	for _, image := range imgs {
		cImage := CloudImage{
			Name:   *image.Name,
			Status: *(*image.ImageProperties).ProvisioningState,
		}

		cimages = append(cimages, cImage)
	}

	return cimages, nil
}

// ListImages lists images on azure
func (a *Azure) ListImages(ctx *Context) error {

	cimages, err := a.GetImages(ctx)
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
		row = append(row, "")
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteImage deletes image from Azure
func (a *Azure) DeleteImage(ctx *Context, imagename string) error {
	imagesClient := a.getImagesClient()

	fut, err := imagesClient.Delete(context.TODO(), a.groupName, imagename)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v", fut)

	return nil
}

// SyncImage syncs image from provider to another provider
func (a *Azure) SyncImage(config *Config, target Provider, image string) error {
	fmt.Println("not yet implemented")
	return nil
}

// CreateInstance - Creates instance on azure Platform
//
// this is kind of a pita
// you have to create the following:
// {vnet, nsg, subnet, ip, nic} before creating the vm
//
// unfortunately this is going to take some serious re-factoring later
// on w/these massive assumptions being put into place
func (a *Azure) CreateInstance(ctx *Context) error {
	username := "fake"
	password := "fake"

	c := ctx.config
	bucket := c.CloudConfig.BucketName
	if bucket == "" {
		bucket = os.Getenv("AZURE_STORAGE_ACCOUNT")
	}
	if bucket == "" {
		fmt.Println("AZURE_STORAGE_ACCOUNT should be set otherwise logs can not be retrieved.")
		os.Exit(1)
	}
	location := a.getLocation(ctx.config)

	debug := false

	vmName := ctx.config.CloudConfig.ImageName + strconv.FormatInt(time.Now().Unix(), 10)
	fmt.Printf("spinning up:\t%s\n", vmName)

	// create virtual network
	vnet, err := a.CreateVirtualNetwork(context.TODO(), location, vmName)
	if err != nil {
		fmt.Println(err)
	}

	if debug {
		fmt.Printf("%+v\n", vnet)
	}

	// create nsg
	nsg, err := a.CreateNetworkSecurityGroup(context.TODO(), location, vmName, c)
	if err != nil {
		fmt.Println(err)
	}

	if debug {
		fmt.Printf("%+v\n", nsg)
	}

	// create subnet
	subnet, err := a.CreateSubnetWithNetworkSecurityGroup(context.TODO(), vmName, vmName, "10.0.0.0/24", vmName)
	if err != nil {
		fmt.Println(err)
	}

	if debug {
		fmt.Printf("%+v\n", subnet)
	}

	// create ip
	ip, err := a.CreatePublicIP(context.TODO(), location, vmName)
	if err != nil {
		fmt.Println(err)
	}

	if debug {
		fmt.Printf("%+v\n", ip)
	}

	// create nic
	// pass vnet, subnet, ip, nicname
	nic, err := a.CreateNIC(context.TODO(), location, vmName, vmName, vmName, vmName, vmName)
	if err != nil {
		fmt.Println(err)
	}

	var sshKeyData string
	sshKeyData = fakepubkey
	nctx := context.TODO()

	fmt.Println("\ncreating the vm - this can take a few minutes - you can ctrl-c this after a bit")
	fmt.Println("there is a known issue that prevents the deploy from ever being 'done'")

	vmClient := a.getVMClient()
	future, err := vmClient.CreateOrUpdate(
		nctx,
		a.groupName,
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(location),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesStandardA1V2,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						ID: to.StringPtr("/subscriptions/" + a.subID + "/resourceGroups/" + a.groupName + "/providers/Microsoft.Compute/images/" + ctx.config.CloudConfig.ImageName),
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
		fmt.Printf("cannot create vm: %v\n", err.Error())
		os.Exit(1)
	}

	err = future.WaitForCompletionRef(nctx, vmClient.Client)
	if err != nil {
		fmt.Printf("cannot get the vm create or update future response: %v\n", err.Error())
		os.Exit(1)
	}

	vm, err := future.Result(vmClient)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v\n", vm)

	if ctx.config.RunConfig.DomainName != "" {
		err = CreateDNSRecord(ctx.config, *ip.IPAddress, a)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (a *Azure) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	vmClient := a.getVMClient()

	vm, err := vmClient.Get(context.TODO(), a.groupName, id, compute.InstanceView)
	if err != nil {
		return nil, err
	}

	return a.convertToCloudInstance(&vm), nil
}

// GetInstances return all instances on Azure
func (a *Azure) GetInstances(ctx *Context) ([]CloudInstance, error) {
	var cinstances []CloudInstance

	vmClient := a.getVMClient()

	vmlist, err := vmClient.List(context.TODO(), a.groupName)
	if err != nil {
		fmt.Println(err)
	}

	instances := vmlist.Values()

	for _, instance := range instances {
		cinstance := a.convertToCloudInstance(&instance)

		cinstances = append(cinstances, *cinstance)
	}

	return cinstances, nil
}

func (a *Azure) convertToCloudInstance(instance *compute.VirtualMachine) *CloudInstance {
	cinstance := CloudInstance{
		Name: *instance.Name,
	}
	privateIP := ""
	publicIP := ""

	if instance.VirtualMachineProperties != nil {
		nifs := *((*(*instance.VirtualMachineProperties).NetworkProfile).NetworkInterfaces)

		for i := 0; i < len(nifs); i++ {
			nicClient := a.getNicClient()
			nic, err := nicClient.Get(context.TODO(), a.groupName, cinstance.Name, "")
			if err != nil {
				fmt.Println(err)
			}

			if nic.InterfacePropertiesFormat != nil {
				ipconfig := *(*nic.InterfacePropertiesFormat).IPConfigurations
				for x := 0; x < len(ipconfig); x++ {
					format := *ipconfig[x].InterfaceIPConfigurationPropertiesFormat
					privateIP = *format.PrivateIPAddress

					ipClient := a.getIPClient()
					pubip, err := ipClient.Get(context.TODO(), a.groupName, cinstance.Name, "")
					if err != nil {
						fmt.Println(err)
					}
					publicIP = *(*pubip.PublicIPAddressPropertiesFormat).IPAddress
				}
			}

		}
	}
	cinstance.PrivateIps = []string{privateIP}
	cinstance.PublicIps = []string{publicIP}

	return &cinstance
}

// ListInstances lists instances on Azure
func (a *Azure) ListInstances(ctx *Context) error {
	cinstances, err := a.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
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
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		table.Append(rows)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from Azure
func (a *Azure) DeleteInstance(ctx *Context, instancename string) error {

	vmClient := a.getVMClient()
	fmt.Printf("\nStarted deleting Instance")
	future, err := vmClient.Delete(context.TODO(), a.groupName, instancename)

	if err != nil {
		exitWithError("\nUnable to delete instance")
	}

	err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
	if err != nil {
		fmt.Printf("\ncannot delete vm future response: %v\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("\nInstance deletion completed")
	return nil
}

// StartInstance starts an instance in Azure
func (a *Azure) StartInstance(ctx *Context, instancename string) error {

	vmClient := a.getVMClient()
	fmt.Printf("Starting instance %s", instancename)
	_, err := vmClient.Start(context.TODO(), a.groupName, instancename)
	if err != nil {
		fmt.Printf("cannot start vm: %v\n", err.Error())
		os.Exit(1)
	}

	return nil
}

// StopInstance deletes instance from Azure
func (a *Azure) StopInstance(ctx *Context, instancename string) error {

	vmClient := a.getVMClient()
	// skipShutdown parameter is optional, we are taking its default
	// value here
	fmt.Printf("Stopping instance %s", instancename)
	_, err := vmClient.PowerOff(context.TODO(), a.groupName, instancename, nil)
	if err != nil {
		fmt.Printf("cannot power off vm: %v\n", err.Error())
		os.Exit(1)
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (a *Azure) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := a.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (a *Azure) GetInstanceLogs(ctx *Context, instancename string) (string, error) {
	// this is basically 2 calls
	// 1) grab the log location
	// 2) grab it from storage

	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		fmt.Println("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		fmt.Printf("Invalid credentials with error: %s\n", err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/", accountName))

	containerURL := azblob.NewContainerURL(*URL, p)

	vmName := instancename

	vmClient := a.getVMClient()
	vm, err := vmClient.Get(context.TODO(), a.groupName, vmName, compute.InstanceView)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

// ResizeImage is not supported on azure.
func (a *Azure) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	return fmt.Errorf("Operation not supported")
}

// GetStorage returns storage interface for cloud provider
func (a *Azure) GetStorage() Storage {
	return a.Storage
}
