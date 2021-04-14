package provider

import (
	"fmt"

	"github.com/nanovms/ops/aws"
	"github.com/nanovms/ops/azure"
	"github.com/nanovms/ops/digitalocean"
	"github.com/nanovms/ops/gcp"
	"github.com/nanovms/ops/hyperv"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/oci"
	"github.com/nanovms/ops/onprem"
	"github.com/nanovms/ops/openstack"
	"github.com/nanovms/ops/types"
	"github.com/nanovms/ops/upcloud"
	"github.com/nanovms/ops/vbox"
	"github.com/nanovms/ops/vsphere"
	"github.com/nanovms/ops/vultr"
)

func CloudProvider(providerName string, c *types.ProviderConfig) (lepton.Provider, error) {
	var provider lepton.Provider

	switch providerName {
	case "gcp":
		provider = gcp.NewGCloud()
	case "onprem":
		provider = &onprem.OnPrem{}
	case "aws":
		provider = &aws.AWS{}
	case "do":
		provider = &digitalocean.DigitalOcean{}
	case "vultr":
		provider = &vultr.Vultr{}
	case "vsphere":
		provider = &vsphere.Vsphere{}
	case "openstack":
		provider = &openstack.OpenStack{}
	case "azure":
		provider = &azure.Azure{}
	case "hyper-v":
		provider = hyperv.NewProvider()
	case "upcloud":
		provider = upcloud.NewProvider()
	case "oci":
		provider = oci.NewProvider()
	case "vbox":
		provider = vbox.NewProvider()
	default:
		return provider, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := provider.Initialize(c)
	return provider, err
}
