package cmd

import (
	"fmt"

	api "github.com/nanovms/ops/lepton"
)

// TODO : use factory or DI
func getCloudProvider(providerName string, config *api.ProviderConfig) (api.Provider, error) {
	var provider api.Provider

	switch providerName {
	case "gcp":
		provider = api.NewGCloud()
	case "onprem":
		provider = &api.OnPrem{}
	case "aws":
		provider = &api.AWS{}
	case "do":
		provider = &api.DigitalOcean{}
	case "vultr":
		provider = &api.Vultr{}
	case "vsphere":
		provider = &api.Vsphere{}
	case "openstack":
		provider = &api.OpenStack{}
	case "azure":
		provider = &api.Azure{}
	default:
		return provider, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := provider.Initialize(config)
	return provider, err
}

func getProviderAndContext(c *api.Config, providerName string) (api.Provider, *api.Context, error) {
	p, err := getCloudProvider(providerName, &c.CloudConfig)
	if err != nil {
		return nil, nil, err
	}

	ctx := api.NewContext(c)

	return p, ctx, nil
}
