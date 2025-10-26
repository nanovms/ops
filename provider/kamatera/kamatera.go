//go:build kamatera || !onlyprovider

package kamatera

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "kamatera"

// Kamatera Provider to interact with Kamatera cloud infrastructure
type Kamatera struct {
	Storage *ObjectStorage
	apiKey  string
}

// NewProvider Kamatera
func NewProvider() *Kamatera {
	return &Kamatera{}
}

type AuthResponse struct {
	Authentication string `json:"authentication"`
	Expires        int    `json:"expires"`
}

func (h *Kamatera) newClient() string {
	url := "https://console.kamatera.com/service/authenticate"

	clientId := os.Getenv("CLIENT_ID")
	secret := os.Getenv("CLIENT_SECRET")

	payload := strings.NewReader(`{"clientId":"` + clientId + `","secret":"` + secret + `"}`)
	client := &http.Client{}

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	var ar AuthResponse

	err = json.Unmarshal(body, &ar)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return ar.Authentication
}

// Initialize Kamatera client
func (h *Kamatera) Initialize(c *types.ProviderConfig) error {
	var err error

	h.apiKey = h.newClient()

	h.Storage = &ObjectStorage{}
	return err
}

// GetStorage returns storage interface for cloud provider
func (h *Kamatera) GetStorage() lepton.Storage {
	return h.Storage
}

func (h *Kamatera) ensureStorage() {
	if h.Storage == nil {
		h.Storage = &ObjectStorage{}
	}
}
