package vultr

import (
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// Vultr provides access to the Vultr API.
type Vultr struct {
	Storage *Objects
}

type vultrSnap struct {
	SnapShotID  string `json:"SNAPSHOTID"`
	CreatedAt   string `json:"date_created"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Status      string `json:"status"`
	OSID        string `json:"OSID"`
	APPID       string `json:"APPID"`
}

type vultrServer struct {
	SUBID     string `json:"SUBID"`
	Status    string `json:"status"`
	PublicIP  string `json:"main_ip"`
	PrivateIP string `json:"internal_ip"`
	CreatedAt string `json:"date_created"`
	Name      string `json:"label"`
}

// Initialize GCP related things
func (v *Vultr) Initialize(config *types.ProviderConfig) error {
	return nil
}

// GetStorage returns storage interface for cloud provider
func (v *Vultr) GetStorage() lepton.Storage {
	return v.Storage
}
