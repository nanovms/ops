package lepton

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// APIKeyHeader is the header key where we set the api key for packagehub
const APIKeyHeader = "x-api-key"

// CredentialFileName is the name of the file which stores packagehub's credentials
const CredentialFileName = "credentials"

// ErrCredentialsNotExist is the error we return if the credential file doesn't exist
var ErrCredentialsNotExist = errors.New("credentials not exist")

// ValidateSuccessResponse is the structure of the success response from validate api key endpoint
type ValidateSuccessResponse struct {
	Username string
}

// Credentials is the information that will be stored in the ~/.ops/credentials
type Credentials struct {
	Username string
	APIKey   string
}

// ValidateAPIKey uses the api key provided and sends it to validate endpoint of packagehub
// if the response is a valid user then we return that or else error is returned.
func ValidateAPIKey(apikey string) (*ValidateSuccessResponse, error) {
	apikeyEndpoint := PkghubBaseURL + "/apikeys/validate"
	req, err := BaseHTTPRequest("POST", apikeyEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(APIKeyHeader, apikey)
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

// ReadCredsFromLocal gets the credentials from the credential file in the ops home
// returns an ErrCredentialsNotExist if the file doesn't exist. This means the user
// hasn't authenticated yet.
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
