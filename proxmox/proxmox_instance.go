package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

// NextIDResponse contains the next available id.
type NextIDResponse struct {
	Data string `json:"data"`
}

func (p *ProxMox) getNextID() string {
	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/cluster/nextid", nil)
	if err != nil {
		fmt.Println(err)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = p.CheckResultType(body, "getnextid")
	if err != nil {
		return ""
	}

	ir := &NextIDResponse{}
	json.Unmarshal([]byte(body), ir)

	return ir.Data
}

// CreateInstance - Creates instance on Proxmox.
func (p *ProxMox) CreateInstance(ctx *lepton.Context) error {
	config := ctx.Config()

	nextid := p.getNextID()

	imageName := config.CloudConfig.ImageName

	data := url.Values{}
	data.Set("vmid", nextid)
	data.Set("name", imageName)
	data.Set("net0", "model=virtio,bridge=vmbr0")

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu", bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "createinstance")
	if err != nil {
		return err
	}

	err = p.addVirtioDisk(ctx, nextid, imageName)
	if err != nil {
		return err
	}

	err = p.movDisk(ctx, nextid, imageName)

	return err
}

func (p *ProxMox) movDisk(ctx *lepton.Context, vmid string, imageName string) error {

	var err error

	data := url.Values{}
	data.Set("disk", "virtio0")
	data.Set("node", p.nodeNAME)
	data.Set("format", "raw")
	data.Set("storage", "local-lvm")
	data.Set("vmid", vmid)

	err = p.CheckStorage("local-lvm")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/move_disk", bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "movdisk")
	if err != nil {
		return err
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	return nil
}

func (p *ProxMox) addVirtioDisk(ctx *lepton.Context, vmid string, imageName string) error {
	data := url.Values{}

	// attach disk
	data.Set("virtio0", "file=local:iso/"+imageName+".iso")

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/config", bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "addvirtiodisk")
	if err != nil {
		return err
	}

	// set boot order, needs to come after attaching disk
	data.Set("boot", "order=virtio0")

	req, err = http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/config", bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Println(err)
		return err
	}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = p.CheckResultType(body, "bootorderset")
	if err != nil {
		return err
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (p *ProxMox) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, errors.New("currently not available")
}

// GetInstances return all instances on ProxMox
func (p *ProxMox) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	var cloudInstances []lepton.CloudInstance
	return cloudInstances, nil
}

// InstanceResponse holds a list of instances info.
type InstanceResponse struct {
	Data []InstanceInfo `json:"data"`
}

// InstanceInfo is a single response type of Instance.
type InstanceInfo struct {
	Name   string `json:"name"`
	VMID   int    `json:"vmid"`
	Status string `json:"status"`
}

// ListInstances lists instances on Proxmox.
func (p *ProxMox) ListInstances(ctx *lepton.Context) error {

	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu", nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	ir := &InstanceResponse{}
	json.Unmarshal([]byte(body), ir)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "MainIP", "Status", "ImageID"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range ir.Data {
		var row []string
		row = append(row, strconv.Itoa(instance.VMID))
		row = append(row, instance.Name)
		row = append(row, "")
		row = append(row, instance.Status)
		row = append(row, "")
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from Proxmox.
func (p *ProxMox) DeleteInstance(ctx *lepton.Context, instanceID string) error {

	req, err := http.NewRequest("DELETE", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+instanceID, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err

	}

	return err

}

// StartInstance starts an instance in Proxmox
func (p *ProxMox) StartInstance(ctx *lepton.Context, instanceID string) error {

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+instanceID+"/status/start", nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// StopInstance halts instance from Proxmox.
func (p *ProxMox) StopInstance(ctx *lepton.Context, instanceID string) error {

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+instanceID+"/status/stop", nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *ProxMox) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not available")
}

// GetInstanceLogs gets instance related logs
func (p *ProxMox) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}
