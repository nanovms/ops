package oci

import (
	"context"

	"github.com/nanovms/ops/types"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-go-sdk/objectstorage"
	"github.com/oracle/oci-go-sdk/workrequests"
	"github.com/spf13/afero"
)

var (
	ociOpsTags = map[string]string{"CreatedBy": "OPS"}
)

func checkHasOpsTags(tags map[string]string) bool {
	val, ok := tags["CreatedBy"]
	return ok && val == "OPS"
}

// ComputeService has OCI client methods to manage images and instances listing
type ComputeService interface {
	CreateImage(ctx context.Context, request core.CreateImageRequest) (response core.CreateImageResponse, err error)
	ListImages(ctx context.Context, request core.ListImagesRequest) (response core.ListImagesResponse, err error)
	DeleteImage(ctx context.Context, request core.DeleteImageRequest) (response core.DeleteImageResponse, err error)
	ListInstances(ctx context.Context, request core.ListInstancesRequest) (response core.ListInstancesResponse, err error)
	LaunchInstance(ctx context.Context, request core.LaunchInstanceRequest) (response core.LaunchInstanceResponse, err error)
	TerminateInstance(ctx context.Context, request core.TerminateInstanceRequest) (response core.TerminateInstanceResponse, err error)
	InstanceAction(ctx context.Context, request core.InstanceActionRequest) (response core.InstanceActionResponse, err error)
	AttachVolume(ctx context.Context, request core.AttachVolumeRequest) (response core.AttachVolumeResponse, err error)
	DetachVolume(ctx context.Context, request core.DetachVolumeRequest) (response core.DetachVolumeResponse, err error)
	ListVnicAttachments(ctx context.Context, request core.ListVnicAttachmentsRequest) (response core.ListVnicAttachmentsResponse, err error)
}

// NetworkService has OCI client methods to manage network instances
type NetworkService interface {
	ListVcns(ctx context.Context, request core.ListVcnsRequest) (response core.ListVcnsResponse, err error)
	ListSubnets(ctx context.Context, request core.ListSubnetsRequest) (response core.ListSubnetsResponse, err error)
	GetVnic(ctx context.Context, request core.GetVnicRequest) (response core.GetVnicResponse, err error)
	CreateNetworkSecurityGroup(ctx context.Context, request core.CreateNetworkSecurityGroupRequest) (response core.CreateNetworkSecurityGroupResponse, err error)
	AddNetworkSecurityGroupSecurityRules(ctx context.Context, request core.AddNetworkSecurityGroupSecurityRulesRequest) (response core.AddNetworkSecurityGroupSecurityRulesResponse, err error)
}

// WorkRequestService has OCI client methods to get work request status
type WorkRequestService interface {
	GetWorkRequest(ctx context.Context, request workrequests.GetWorkRequestRequest) (response workrequests.GetWorkRequestResponse, err error)
}

// StorageService has OCI client methods to manage storage block, required to upload images
type StorageService interface {
	PutObject(ctx context.Context, request objectstorage.PutObjectRequest) (response objectstorage.PutObjectResponse, err error)
}

// BlockstorageService has OCI client methods to manage volumes
type BlockstorageService interface {
	CreateVolume(ctx context.Context, request core.CreateVolumeRequest) (response core.CreateVolumeResponse, err error)
	ListVolumes(ctx context.Context, request core.ListVolumesRequest) (response core.ListVolumesResponse, err error)
	DeleteVolume(ctx context.Context, request core.DeleteVolumeRequest) (response core.DeleteVolumeResponse, err error)
}

// Provider has methods to interact with oracle cloud infrastructure
type Provider struct {
	computeClient      ComputeService
	storageClient      StorageService
	workRequestClient  WorkRequestService
	networkClient      NetworkService
	blockstorageClient BlockstorageService
	fileSystem         afero.Fs
	compartmentID      string
	availabilityDomain string
}

// NewProvider returns an instance of OCI Provider
func NewProvider() *Provider {
	return &Provider{
		computeClient: nil,
		storageClient: nil,
		fileSystem:    afero.NewOsFs(),
	}
}

// NewProviderWithClients returns an instance of OCI Provider with required clients initialized
func NewProviderWithClients(c ComputeService, s StorageService, w WorkRequestService, n NetworkService, b BlockstorageService, f afero.Fs) *Provider {
	return &Provider{c, s, w, n, b, f, "", ""}
}

// Initialize checks conditions to use oci
func (p *Provider) Initialize(providerConfig *types.ProviderConfig) (err error) {
	config := common.DefaultConfigProvider()

	p.computeClient, err = core.NewComputeClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	p.storageClient, err = objectstorage.NewObjectStorageClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	p.workRequestClient, err = workrequests.NewWorkRequestClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	p.networkClient, err = core.NewVirtualNetworkClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	p.blockstorageClient, err = core.NewBlockstorageClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	p.compartmentID, _ = config.TenancyOCID()

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(config)
	if err != nil {
		return
	}

	domains, err := identityClient.ListAvailabilityDomains(context.TODO(), identity.ListAvailabilityDomainsRequest{CompartmentId: &p.compartmentID})
	if err != nil {
		return
	}

	if len(domains.Items) > 0 {
		p.availabilityDomain = *domains.Items[0].Name
	}

	return
}
