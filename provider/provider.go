package provider

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/aws"
	"github.com/nanovms/ops/provider/azure"
	"github.com/nanovms/ops/provider/digitalocean"
	"github.com/nanovms/ops/provider/gcp"
	"github.com/nanovms/ops/provider/hyperv"
	"github.com/nanovms/ops/provider/oci"
	"github.com/nanovms/ops/provider/onprem"
	"github.com/nanovms/ops/provider/openshift"
	"github.com/nanovms/ops/provider/openstack"
	"github.com/nanovms/ops/provider/proxmox"
	"github.com/nanovms/ops/provider/upcloud"
	"github.com/nanovms/ops/provider/vbox"
	"github.com/nanovms/ops/provider/vsphere"
	"github.com/nanovms/ops/provider/vultr"
	"github.com/nanovms/ops/types"
)

// CloudProvider is a factory that returns an existing provider based on provider type passed by argument
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
	case "proxmox":
		provider = &proxmox.ProxMox{}
	case "openshift":
		provider = openshift.NewProvider()
	default:
		return provider, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := provider.Initialize(c)
	return provider, err
}
