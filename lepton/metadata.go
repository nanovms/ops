package lepton

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/nanovms/ops/log"
)

type APIMetadataRequest struct {
	Namespace string `json:"namespace"`
	PkgName   string `json:"pkg_name"`
	Version   string `json:"version"`
}

// GetPackageMetadata get metadata for the package
func GetPackageMetadata(namespace, pkgName, version string) (*Package, error) {
	var err error

	creds, err := ReadCredsFromLocal()
	if err != nil {
		if err == ErrCredentialsNotExist {
			// for a better error message
			log.Fatal(errors.New("user is not logged in. use 'ops pkg login' first"))
		} else {
			log.Fatal(err)
		}
	}

	// this would never error out
	metadataURL, _ := url.Parse(PkghubBaseURL + "/api/v1/pkg/metadata")
	data, err := json.Marshal(APIMetadataRequest{
		Namespace: namespace,
		PkgName:   pkgName,
		Version:   version,
	})
	if err != nil {
		log.Fatal(err)
	}
	req, err := BaseHTTPRequest("POST", metadataURL.String(), bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set(APIKeyHeader, creds.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var pkg Package
	err = json.NewDecoder(resp.Body).Decode(&pkg)
	return &pkg, err
}
