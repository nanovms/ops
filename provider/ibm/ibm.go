package ibm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

// ProviderName of the cloud platform provider
const ProviderName = "ibm"

// IBM Provider to interact with IBM infrastructure
type IBM struct {
	Storage *Objects
	token   string
	iam     string
}

// NewProvider IBM
func NewProvider() *IBM {
	return &IBM{}
}

// Initialize provider
func (v *IBM) Initialize(config *types.ProviderConfig) error {
	v.token = os.Getenv("TOKEN")
	if v.token == "" {
		return fmt.Errorf("TOKEN is not set")
	}

	v.setIAMToken()
	return nil
}

// Token is the return type for a new IAM token.
type Token struct {
	AccessToken string `json:"access_token"`
}

func (v *IBM) setIAMToken() {

	uri := "https://iam.cloud.ibm.com/oidc/token"

	data := url.Values{}
	data.Set("apikey", v.token)
	data.Set("response_type", "cloud_iam")
	data.Set("grant_type", "urn:ibm:params:oauth:grant-type:apikey")

	client := &http.Client{}
	r, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println(err)
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Accept", "application/json")

	res, err := client.Do(r)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	it := &Token{}
	err = json.Unmarshal(body, &it)
	if err != nil {
		fmt.Println(err)
	}

	v.iam = it.AccessToken
}

// GetStorage returns storage interface for cloud provider
func (v *IBM) GetStorage() lepton.Storage {
	return v.Storage
}
