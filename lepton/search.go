package lepton

import (
	"encoding/json"
	"net/http"
	"net/url"
)

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
