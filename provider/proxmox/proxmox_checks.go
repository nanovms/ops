//go:build proxmox || !onlyprovider

package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type pData struct {
	Data string `json:"data"`
}

type pDataArr struct {
	Data []string `json:"data"`
}

type sData struct {
	Active  int    `json:"active"`
	Enabled int    `json:"enabled"`
	Storage string `json:"storage"`
	Content string `json:"content"`
}

type sDataArr struct {
	Data []sData `json:"data"`
}

type bData struct {
	Active int    `json:"active"`
	Type   string `json:"type"`
	Iface  string `json:"iface"`
}

type bDataArr struct {
	Data []bData `json:"data"`
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

	body, err := io.ReadAll(resp.Body)
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

// CheckStorage return error when not found configured storage or any storages via ProxMox API
func (p *ProxMox) CheckStorage(storage string, stype string) error {

	var err error

	var sdp sDataArr

	var b bytes.Buffer

	edk := errors.New("no any storages is configured")
	enb := errors.New("storage is disabled: " + storage)
	ect := errors.New("storage is not active: " + storage)
	ecs := errors.New("not found storage: " + storage)
	eim := errors.New("storage is not configured for containing disk images: " + storage)
	eis := errors.New("storage is not configured for containing iso images: " + storage)

	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/storage", &b)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &sdp)
	if err != nil {
		return err
	}

	if err == nil && sdp.Data == nil {
		return edk
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	for _, v := range sdp.Data {

		if v.Storage == storage {

			if v.Active != 1 {
				return ect
			}

			if v.Enabled != 1 {
				return enb
			}

			if stype == "images" {

				if !strings.Contains(v.Content, stype) {
					return eim
				}

			} else if stype == "iso" {

				if !strings.Contains(v.Content, stype) {
					return eis
				}

			} else {
				return errors.New("unknown type of storage")
			}

			return nil

		}
	}

	return ecs

}

// CheckBridge return error when not found configured bridge any network interfaces via ProxMox API
func (p *ProxMox) CheckBridge(bridge string) error {

	var err error

	var brs bDataArr

	var b bytes.Buffer

	edk := errors.New("no any bridges is configured")
	ebr := errors.New("is not a bridge: " + bridge)
	ect := errors.New("bridge is not active: " + bridge)
	ecs := errors.New("not found bridge: " + bridge)

	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/network", &b)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &brs)
	if err != nil {
		return err
	}

	if err == nil && brs.Data == nil {
		return edk
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	for _, v := range brs.Data {

		if v.Iface == bridge {

			if v.Active != 1 {
				return ect
			}

			if v.Type != "bridge" {
				return ebr
			}

			return nil

		}
	}

	return ecs

}

// CheckResult return error or custom error when {"data": null} or {"data": []} result come from ProxMox API (Not used yet)
func (p *ProxMox) CheckResult(body []byte) error {

	var err error

	ecs := errors.New("check token permissions, environment variables correctness")

	err = checkErrs(body)
	if err != nil {
		return err
	}

	err = checkData(body)
	if err != nil {
		return ecs
	}

	return nil

}

// CheckResultType return error or custom error based on type of check, when {"data": null} or {"data": []} result come from ProxMox API
func (p *ProxMox) CheckResultType(body []byte, rtype string, rname string) error {

	var err error

	edef := "check token permissions, environment variables correctness"
	ecs := errors.New(edef)

	switch rtype {
	case "createimage":
		ecs = errors.New("can not create disk image in " + rname + " storage, " + edef)
	case "listimages":
		ecs = errors.New("no disk images found in " + rname + " storage or " + edef)
	case "createinstance":
		ecs = errors.New("can not create machine instance with disk image from " + rname + " image/storage, " + edef)
	case "getnextid":
		ecs = errors.New("can not get next id, " + edef)
	case "movdisk":
		ecs = errors.New("can not move iso to raw disk on " + rname + " storage, " + edef)
	case "addvirtiodisk":
		ecs = errors.New("can not add virtio disk from " + rname + " storage, " + edef)
	case "bootorderset":
		ecs = errors.New("can not set boot order, " + edef)
	}

	err = checkErrs(body)
	if err != nil {
		return err
	}

	err = checkData(body)
	if err != nil {
		return ecs
	}

	return nil

}

// checkErrs return error from ProxMox API as is
func checkErrs(body []byte) error {

	var err error

	var est map[string]*json.RawMessage

	err = json.Unmarshal(body, &est)
	if err == nil {
		e, ok := est["errors"]
		if ok {
			return errors.New(string(*e))
		}
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
