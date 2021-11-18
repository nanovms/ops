package proxmox

import (
	"fmt"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProxMox provides access to the ProxMox API.
type ProxMox struct {
	Storage *Objects
	tokenID string
	secret  string
	apiURL  string
}

// Initialize provider
func (p *ProxMox) Initialize(config *types.ProviderConfig) error {
	p.tokenID = os.Getenv("TOKEN_ID")
	if p.tokenID == "" {
		return fmt.Errorf("TOKEN_ID is not set")
	}

	p.secret = os.Getenv("SECRET")
	if p.secret == "" {
		return fmt.Errorf("SECRET is not set")
	}

	p.apiURL = os.Getenv("API_URL")
	if p.apiURL == "" {
		return fmt.Errorf("API_URL is not set")
	}

	return nil
}

// GetStorage returns storage interface for cloud provider
func (p *ProxMox) GetStorage() lepton.Storage {
	return p.Storage
}
