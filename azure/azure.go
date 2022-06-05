package azure

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-12-01/compute"
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
	Storage         *Storage
	subID           string
	clientID        string
	tenantID        string
	clientSecret    string
	locationDefault string
	groupName       string
	storageAccount  string
	hyperVGen       compute.HyperVGenerationTypes

	authorizer *autorest.Authorizer
}

func getAzureDefaultTags() map[string]*string {
	return map[string]*string{
		"CreatedBy": to.StringPtr("ops"),
	}
}

func hasAzureOpsTags(tags map[string]*string) bool {
	val, ok := tags["CreatedBy"]

	return ok && *val == "ops"
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

	oauthConfig, err := adal.NewOAuthConfig(a.Environment().ActiveDirectoryEndpoint, a.tenantID)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, a.clientID, a.clientSecret, resource)
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

func (a *Azure) getImagesClient() *compute.ImagesClient {
	vmClient := compute.NewImagesClientWithBaseURI(compute.DefaultBaseURI, a.subID)
	vmClient.Authorizer = *a.authorizer
	vmClient.AddToUserAgent(userAgent)
	return &vmClient
}

func (a *Azure) getVMClient() *compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(a.subID)
	vmClient.Authorizer = *a.authorizer
	vmClient.AddToUserAgent(userAgent)
	return &vmClient
}

func (a *Azure) getVMExtensionsClient() compute.VirtualMachineExtensionsClient {
	extClient := compute.NewVirtualMachineExtensionsClient(a.subID)
	extClient.Authorizer = *a.authorizer
	extClient.AddToUserAgent(userAgent)
	return extClient
}

func (a *Azure) getLocation(config *types.Config) string {
	c := config
	location := c.CloudConfig.Zone
	if location == "" {
		location = a.locationDefault
	}
	if location == "" {
		log.Fatalf("Error: a location must be set via either the Zone attribute in CloudConfig or the AZURE_LOCATION_DEFAULT environment variable.")
	}
	return location
}

func stripAndExtractAvailibilityZone(location string) (string, string) {
	exp, err := regexp.Compile(`\-\d$`)

	// this error wouldn't happen as we control the regex
	if err != nil {
		log.Fatalf("error while extracting availibility zone")
	}
	if exp.Match([]byte(location)) {
		// returns the last part of the location as az
		az := location[len(location)-1:]
		// strips the "-[1-9]" from the end of the location if its present
		cleanedLoc := location[:len(location)-2]
		return cleanedLoc, az
	}
	return location, ""
}

// Initialize Azure related things
func (a *Azure) Initialize(config *types.ProviderConfig) error {
	a.Storage = &Storage{}

	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subID != "" {
		a.subID = subID
	} else {
		return fmt.Errorf("set AZURE_SUBSCRIPTION_ID")
	}

	locationDefault := os.Getenv("AZURE_LOCATION_DEFAULT")
	if locationDefault != "" {
		a.locationDefault = locationDefault
	}

	clientID := os.Getenv("AZURE_CLIENT_ID")
	if clientID != "" {
		a.clientID = strings.TrimSpace(clientID)
	} else {
		return fmt.Errorf("set AZURE_CLIENT_ID")
	}

	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if clientSecret != "" {
		a.clientSecret = strings.TrimSpace(clientSecret)
	} else {
		return fmt.Errorf("set AZURE_CLIENT_SECRET")
	}

	tenantID := os.Getenv("AZURE_TENANT_ID")
	if tenantID != "" {
		a.tenantID = strings.TrimSpace(tenantID)
	} else {
		return fmt.Errorf("set AZURE_TENANT_ID")
	}

	groupName := os.Getenv("AZURE_BASE_GROUP_NAME")
	if groupName != "" {
		a.groupName = strings.TrimSpace(groupName)
	} else {
		return fmt.Errorf("set AZURE_BASE_GROUP_NAME")
	}

	storageAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if storageAccount != "" {
		a.storageAccount = strings.TrimSpace(storageAccount)
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	a.authorizer = &authorizer

	return nil
}

func (a *Azure) getBucketName() (string, error) {
	if a.storageAccount != "" {
		return a.storageAccount, nil
	}

	return "", errors.New("AZURE_STORAGE_ACCOUNT should be set")
}

// GetStorage returns storage interface for cloud provider
func (a *Azure) GetStorage() lepton.Storage {
	return a.Storage
}
