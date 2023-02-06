package provider

import (
	"fmt"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"

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
)

// CloudProvider is a factory that returns an existing provider based on provider type passed by argument
func CloudProvider(providerName string, c *types.ProviderConfig) (lepton.Provider, error) {
	var p lepton.Provider

	switch providerName {
	case aws.ProviderName:
		p = aws.NewProvider()

	case azure.ProviderName:
		p = azure.NewProvider()

	case digitalocean.ProviderName:
		p = digitalocean.NewProvider()

	case gcp.ProviderName:
		p = gcp.NewProvider()

	case hyperv.ProviderName:
		p = hyperv.NewProvider()

	case oci.ProviderName:
		p = oci.NewProvider()

	case onprem.ProviderName:
		p = onprem.NewProvider()

	case openshift.ProviderName:
		p = openshift.NewProvider()

	case openstack.ProviderName:
		p = openstack.NewProvider()

	case proxmox.ProviderName:
		p = proxmox.NewProvider()

	case upcloud.ProviderName:
		p = upcloud.NewProvider()

	case vbox.ProviderName:
		p = vbox.NewProvider()

	case vsphere.ProviderName:
		p = vsphere.NewProvider()

	case vultr.ProviderName:
		p = vultr.NewProvider()

	default:
		return p, fmt.Errorf("error:Unknown provider %s", providerName)
	}

	err := p.Initialize(c)
	return p, err
}
