package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type pData struct {
	Data string `json:"data"`
}

type pDataArr struct {
	Data []string `json:"data"`
}

// CheckInit return custom error on {"data": null} or {"data": []} result come from ProxMox API /api2/json/pools
func (p *ProxMox) CheckInit() error {

	var err error

	var b bytes.Buffer

	ecs := errors.New("check token permissions, environment variables correctness, having at least one pool in proxmox")

	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/pools", &b)
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = checkData(body)
	if err != nil {
		return ecs
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	return nil

}

// CheckResult return custom error when {"data": null} or {"data": []} result come from ProxMox API (Not used now)
func (p *ProxMox) CheckResult(body []byte) error {

	var err error

	ecs := errors.New("check token permissions, environment variables correctness")

	err = checkData(body)
	if err != nil {
		return ecs
	}

	return nil

}

// CheckResultType return custom errors based on type of check, when {"data": null} or {"data": []} result come from ProxMox API
func (p *ProxMox) CheckResultType(body []byte, rtype string) error {

	var err error

	edef := "check token permissions, environment variables correctness"
	ecs := errors.New(edef)

	switch rtype {
	case "createimage":
		ecs = errors.New("can not create disk image in 'local' storage, " + edef)
	case "listimages":
		ecs = errors.New("no have any disk images in 'local' storage or " + edef)
	case "createinstance":
		ecs = errors.New("can not create machine instance with disk image from 'local' storage, " + edef)
	case "getnextid":
		ecs = errors.New("can not get next id, " + edef)
	case "movdisk":
		ecs = errors.New("can not move iso to raw disk on 'local-lvm' storage, " + edef)
	case "addvirtiodisk":
		ecs = errors.New("can not add virtio disk from 'local' storage, " + edef)
	case "bootorderset":
		ecs = errors.New("can not set boot order, " + edef)
	}

	err = checkData(body)
	if err != nil {
		return ecs
	}

	return nil

}

// checkData return custom error on {"data": null} or {"data": []} result come from JSON with 'data' string/array field
func checkData(body []byte) error {

	var err error

	var pd pData
	var apd pDataArr

	edk := errors.New("empty data key")

	err = json.Unmarshal(body, &pd)
	if err != nil {
		err = json.Unmarshal(body, &apd)
		if err == nil && apd.Data == nil {
			return edk
		}
	}

	if err == nil && pd.Data == "" {
		return edk
	}

	return nil

}
