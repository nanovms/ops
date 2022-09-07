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

	err = p.CheckResultType(body, "getnextid", "")
	if err != nil {
		return ""
	}

	ir := &NextIDResponse{}
	json.Unmarshal([]byte(body), ir)

	return ir.Data
}

// CreateInstance - Creates instance on Proxmox.
func (p *ProxMox) CreateInstance(ctx *lepton.Context) error {

	var err error

	config := ctx.Config()

	nextid := p.getNextID()

	instanceName := config.RunConfig.InstanceName

	p.isoStorageName = config.TargetConfig["isoStorageName"]

	p.imageName = config.TargetConfig["imageName"]

	p.arch = config.TargetConfig["arch"]
	if p.arch == "" {
		p.arch = "x86_64"
	}

	p.machine = config.TargetConfig["machine"]

	socketsStr := config.TargetConfig["sockets"]
	p.sockets, err = strconv.Atoi(config.TargetConfig["sockets"])
	if err != nil {
		fmt.Println(err)
	}
	if p.sockets == 0 {
		p.sockets = 1
		socketsStr = "1"
	}

	coresStr := ctx.Config().TargetConfig["cores"]
	p.cores, err = strconv.Atoi(ctx.Config().TargetConfig["cores"])
	if err != nil {
		fmt.Println(err)
	}
	if p.cores == 0 {
		p.cores = 1
		coresStr = "1"
	}

	p.numa, err = strconv.ParseBool(ctx.Config().TargetConfig["numa"])
	if err != nil {
		fmt.Println(err)
	}
	numa := "0"
	if p.numa {
		numa = "1"
	}

	p.memory = config.TargetConfig["memory"]
	if p.memory == "" {
		p.memory = "512M"
	}
	memoryInt, err := lepton.RAMInBytes(p.memory)
	if err != nil {
		return err
	}
	memoryInt = memoryInt / 1024 / 1024
	memoryStr := strconv.FormatInt(memoryInt, 10)

	p.storageName = config.TargetConfig["storageName"]
	p.isoStorageName = config.TargetConfig["isoStorageName"]

	onboot := "0"
	p.onboot, err = strconv.ParseBool(config.TargetConfig["onboot"])
	if err != nil {
		fmt.Println(err)
	}
	if p.onboot {
		onboot = "1"
	}

	p.protection, err = strconv.ParseBool(config.TargetConfig["protection"])
	if err != nil {
		fmt.Println(err)
	}
	protectionStr := "0"
	if p.protection {
		protectionStr = "1"
	}

	if p.storageName == "" {
		p.storageName = "local-lvm"
	}

	if p.isoStorageName == "" {
		p.isoStorageName = "local"
	}

	// might be preferential to move some of this to a lint check
	// instead
	err = p.CheckStorage(p.storageName, "images")
	if err != nil {
		return err
	}

	// don't need to pass it here
	err = p.CheckStorage(p.isoStorageName, "iso")
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("vmid", nextid)
	data.Set("name", instanceName)
	if p.machine != "" {
		data.Set("machine", p.machine)
	}
	data.Set("sockets", socketsStr)
	data.Set("cores", coresStr)

	data.Set("numa", numa)
	data.Set("memory", memoryStr)
	data.Set("onboot", onboot)
	data.Set("protection", protectionStr)

	data.Set("name", imageName)

	nics := config.RunConfig.Nics
	for i := 0; i < len(nics); i++ {
		is := strconv.Itoa(i)
		brName := nics[i].BridgeName
		if brName == "" {
			brName = "vmbr" + is
		}

		err = p.CheckBridge(brName)
		if err != nil {
			return err
		}

		data.Set("net"+is, "model=virtio,bridge="+brName)
	}

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

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	err = p.CheckResultType(body, "createinstance", "file="+p.isoStorageName+":iso/"+p.imageName+".iso")
	if err != nil {
		return err
	}

	err = p.addVirtioDisk(ctx, nextid, p.imageName, p.isoStorageName)
	if err != nil {
		return err
	}

	err = p.movDisk(ctx, nextid, p.imageName, p.storageName)

	return err
}

func (p *ProxMox) movDisk(ctx *lepton.Context, vmid string, imageName string, storageName string) error {

	data := url.Values{}
	data.Set("disk", "virtio0")
	data.Set("node", p.nodeNAME)
	data.Set("format", "raw")
	data.Set("storage", storageName)
	data.Set("vmid", vmid)

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

	err = p.CheckResultType(body, "movdisk", storageName)
	if err != nil {
		return err
	}

	debug := false
	if debug {
		fmt.Println(string(body))
	}

	return nil
}

func (p *ProxMox) addVirtioDisk(ctx *lepton.Context, vmid string, imageName string, isoStorageName string) error {

	data := url.Values{}

	// attach disk
	data.Set("virtio0", "file="+isoStorageName+":iso/"+imageName+".iso")

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

	err = p.CheckResultType(body, "addvirtiodisk", isoStorageName)
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

	err = p.CheckResultType(body, "bootorderset", "")
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
