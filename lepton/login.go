package lepton

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const API_KEY_HEADER = "x-api-key"
const CredentialFileName = "credentials"

var ErrCredentialsNotExist = errors.New("credentials not exist")

type ValidateSuccessResponse struct {
	Username string
	Email    string
	GithubID int
}

// Credentials is the information that will be stored in the ~/.ops/credentials
type Credentials struct {
	Username string
	Email    string
	ApiKey   string
}

// ValidateAPIKey uses the api key provided and sends it to validate endpoint of packagehub
// if the response is a valid user then we return that or else error is returned.
func ValidateAPIKey(apikey string) (*ValidateSuccessResponse, error) {
	apikeyEndpoint := PkghubBaseURL + "/apikey/validate"
	req, err := http.NewRequest("POST", apikeyEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(API_KEY_HEADER, apikey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 200 {
		successResp := ValidateSuccessResponse{}
		err = json.NewDecoder(resp.Body).Decode(&successResp)
		if err != nil {
			return nil, err
		}
		return &successResp, nil
	} else if resp.StatusCode == 403 {
		return nil, errors.New("incorrect api-key")
	} else {
		// at this point its mostly a internal server error
		reason, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(reason))
	}

}

// StoreCredentials stores the credentials in the credential file. Overrides it if there is
// an existing one.
func StoreCredentials(creds Credentials) error {
	opsHome := GetOpsHome()
	credentialFilePath := filepath.Join(opsHome, CredentialFileName)
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(credentialFilePath, data, 0644)
}

func ReadCredsFromLocal() (*Credentials, error) {
	opsHome := GetOpsHome()
	credentialFilePath := filepath.Join(opsHome, CredentialFileName)
	if _, err := os.Stat(credentialFilePath); os.IsNotExist(err) {
		return nil, ErrCredentialsNotExist
	}
	data, err := ioutil.ReadFile(credentialFilePath)
	if err != nil {
		return nil, err
	}

	creds := Credentials{}
	if err = json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
