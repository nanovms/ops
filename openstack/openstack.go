package openstack

import (
	"errors"
	"os"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

func getOpenstackOpsTags() []string {
	return []string{"CreatedBy:ops"}
}

// OpenStack provides access to the OpenStack API.
type OpenStack struct {
	provider *gophercloud.ProviderClient
}

// Initialize OpenStack related things
func (o *OpenStack) Initialize(config *types.ProviderConfig) error {

	opts, err := openstack.AuthOptionsFromEnv()

	if err != nil {
		return err
	}

	o.provider, err = openstack.AuthenticatedClient(opts)
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenStack) findFlavorByName(name string) (id string, err error) {
	client, err := o.getComputeClient()
	if err != nil {
		log.Error(err)
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

	return "", errors.New("flavor " + name + " not found")
}

func (o *OpenStack) getComputeClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(o.provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
}
