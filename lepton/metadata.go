package lepton

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"runtime"
)

// APIMetadataRequest is payload sent to get metadata for a package
type APIMetadataRequest struct {
	Namespace string `json:"namespace"`
	PkgName   string `json:"pkg_name"`
	Version   string `json:"version"`
	Arch      string `json:"arch"`
}

// GetPackageMetadata get metadata for the package
func GetPackageMetadata(namespace, pkgName, version string) (*Package, error) {
	var err error

	// we ignore the error here
	creds, _ := ReadCredsFromLocal()

	ar := APIMetadataRequest{
		Namespace: namespace,
		PkgName:   pkgName,
		Version:   version,
	}

	if AltGOARCH != "" {
		if runtime.GOARCH == "arm64" {
			ar.Arch = "" // amd64
		} else {
			ar.Arch = "arm64"
		}
	} else {
		if runtime.GOARCH == "amd64" {
			ar.Arch = "arm64"
		}
	}

	// this would never error out
	metadataURL, _ := url.Parse(PkghubBaseURL + "/api/v1/pkg/metadata")
	data, err := json.Marshal(ar)
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
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("package not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	var pkg Package
	err = json.NewDecoder(resp.Body).Decode(&pkg)
	if err != nil {
		return nil, err
	}
	return &pkg, err
}
