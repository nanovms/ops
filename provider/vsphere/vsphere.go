package vsphere

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	vmwareTypes "github.com/vmware/govmomi/vim25/types"
)

// Vsphere provides access to the Vsphere API.
type Vsphere struct {
	Storage *Datastores
	client  *vim25.Client

	datacenter   string
	datastore    string
	network      string
	resourcePool string
}

// Initialize Vsphere related things
func (v *Vsphere) Initialize(config *types.ProviderConfig) error {
	u, err := v.getCredentials()
	if err != nil {
		return err
	}
	// consume from env vars if set
	dc := os.Getenv("GOVC_DATACENTER")
	ds := os.Getenv("GOVC_DATASTORE")
	nw := os.Getenv("GOVC_NETWORK")
	rp := os.Getenv("GOVC_RESOURCE_POOL")

	v.datacenter = "/ha-datacenter/"
	if dc != "" {
		v.datacenter = dc
	}

	v.datastore = v.datacenter + "datastore/datastore1/"
	if ds != "" {
		v.datastore = ds
	}

	v.network = v.datacenter + "network/VM Network"
	if nw != "" {
		v.network = nw
	}

	// this can be inferred?
	v.resourcePool = v.datacenter + "host/localhost.hsd1.ca.comcast.net/Resources"
	if rp != "" {
		v.resourcePool = rp
	}

	un := u.User.Username()
	pw, _ := u.User.Password()
	soapClient := soap.NewClient(u, true)
	v.client, err = vim25.NewClient(context.Background(), soapClient)
	if err != nil {
		return err
	}

	req := vmwareTypes.Login{
		This: *v.client.ServiceContent.SessionManager,
	}
	req.UserName = un
	req.Password = pw

	_, err = methods.Login(context.Background(), v.client, &req)
	if err != nil {
		return err
	}

	return nil
}

func (v *Vsphere) getCredentials() (*url.URL, error) {
	var tempURL string
	gu := os.Getenv("GOVC_URL")
	if gu == "" {
		return nil, fmt.Errorf("At the very least GOVC_URL should be set to https://host:port")
	}
	// warn if HTTP?
	if !strings.Contains(gu, "http") {
		tempURL = "https://" + gu
	} else {
		tempURL = gu
	}
	u, err := url.Parse(tempURL + "/sdk")
	if err != nil {
		return nil, err
	}

	// if credential is found and not empty string, return immediately
	un := u.User.Username()
	up, ok := u.User.Password()
	if un != "" && up != "" && ok {
		return u, nil
	}

	if un == "" {
		un = os.Getenv("GOVC_USERNAME")
	}
	if un == "" {
		return nil, fmt.Errorf("Incomplete credentials, set either via <GOVC_URL> with https://username:password@host:port or <GOVC_USERNAME and GOVC_PASSWORD>")
	}
	var pw string
	if ok {
		pw = up
	} else {
		pw = os.Getenv("GOVC_PASSWORD")
	}
	if pw == "" {
		return nil, fmt.Errorf("Incomplete credentials, set either via <GOVC_URL> with https://username:password@host:port or <GOVC_USERNAME and GOVC_PASSWORD>")
	}

	tempURL = fmt.Sprintf("%s://%s:%s@%s", u.Scheme, un, url.PathEscape(pw), u.Host)
	u, err = url.Parse(tempURL + "/sdk")
	return u, err
}

// GetStorage returns storage interface for cloud provider
func (v *Vsphere) GetStorage() lepton.Storage {
	return v.Storage
}
