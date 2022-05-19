package lepton

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

type APIMetadataRequest struct {
	Namespace string `json:"namespace"`
	PkgName   string `json:"pkg_name"`
	Version   string `json:"version"`
}

// GetPackageMetadata get metadata for the package
func GetPackageMetadata(namespace, pkgName, version string) (*Package, error) {
	var err error

	// we ignore the error here
	creds, _ := ReadCredsFromLocal()

	// this would never error out
	metadataURL, _ := url.Parse(PkghubBaseURL + "/api/v1/pkg/metadata")
	data, err := json.Marshal(APIMetadataRequest{
		Namespace: namespace,
		PkgName:   pkgName,
		Version:   version,
	})
	if err != nil {
		return nil, err
	}
	req, err := BaseHTTPRequest("POST", metadataURL.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	if creds != nil {
		req.Header.Set(APIKeyHeader, creds.APIKey)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var pkg Package
	err = json.NewDecoder(resp.Body).Decode(&pkg)
	if err != nil {
		return nil, err
	}
	return &pkg, err
}
