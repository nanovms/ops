package lepton

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// SearchPackages searches packages against pkghub
func SearchPackages(q string) (*PackageList, error) {
	// we can ignore this error as the url being parsed here is const so wouldn't error
	pkghub, _ := url.Parse(PkghubBaseURL + "/api/v1/search")
	query := pkghub.Query()
	query.Add("q", q)
	pkghub.RawQuery = query.Encode()

	req, err := BaseHTTPRequest("GET", pkghub.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	pkgList := PackageList{}
	err = json.NewDecoder(response.Body).Decode(&pkgList)
	if err != nil {
		return nil, err
	}
	return &pkgList, nil
}

// SearchPackages searches packages against pkghub
func SearchPackagesWithArch(q string, arch string) (*PackageList, error) {
	// we can ignore this error as the url being parsed here is const so
	// wouldn't error
	pkghub, _ := url.Parse(PkghubBaseURL + "/api/v1/search")
	query := pkghub.Query()
	query.Add("q", q)

	if arch == "amd64" {
		arch = "x86_64"
	}

	query.Add("arch", arch)
	pkghub.RawQuery = query.Encode()

	req, err := BaseHTTPRequest("GET", pkghub.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	pkgList := PackageList{}

	err = json.NewDecoder(response.Body).Decode(&pkgList)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &pkgList, nil
}
