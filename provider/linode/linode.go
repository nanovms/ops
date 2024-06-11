//go:build linode || !onlyprovider

package linode

import (
	"fmt"
	"net/http"
	"os"

	"github.com/linode/linodego"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"golang.org/x/oauth2"
)

// ProviderName of the cloud platform provider
const ProviderName = "linode"

// Linode Provider to interact with Linode infrastructure
type Linode struct {
	Storage *Objects
	Client  linodego.Client
}

// NewProvider Linode
func NewProvider() *Linode {
	return &Linode{}
}

// Initialize provider
func (v *Linode) Initialize(config *types.ProviderConfig) error {
	token := os.Getenv("TOKEN")
	if token == "" {
		return fmt.Errorf("TOKEN is not set")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	v.Client = linodego.NewClient(oauth2Client)

	//	ctx := context.Background()
	return nil
}

// GetStorage returns storage interface for cloud provider
func (v *Linode) GetStorage() lepton.Storage {
	return v.Storage
}
