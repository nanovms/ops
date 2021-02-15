package cmd

import (
	"fmt"

	"github.com/nanovms/ops/config"
	"github.com/nanovms/ops/digitalocean"
	"github.com/nanovms/ops/hyperv"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/upcloud"
)

// TODO : use factory or DI
func getCloudProvider(providerName string, c *config.ProviderConfig) (api.Provider, error) {
	var provider api.Provider

	switch providerName {
	case "gcp":
		provider = api.NewGCloud()
	case "onprem":
		provider = &api.OnPrem{}
	case "aws":
		provider = &api.AWS{}
	case "do":
		provider = &digitalocean.DigitalOcean{}
	case "vultr":
		provider = &api.Vultr{}
	case "vsphere":
		provider = &api.Vsphere{}
	case "openstack":
		provider = &api.OpenStack{}
	case "azure":
		provider = &api.Azure{}
	case "hyper-v":
		provider = hyperv.NewProvider()
	case "upcloud":
		provider = upcloud.NewProvider()
	default:
		return provider, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := provider.Initialize(c)
	return provider, err
}

func getProviderAndContext(c *config.Config, providerName string) (api.Provider, *api.Context, error) {
	p, err := getCloudProvider(providerName, &c.CloudConfig)
	if err != nil {
		return nil, nil, err
	}

	ctx := api.NewContext(c)

	return p, ctx, nil
}
