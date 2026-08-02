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
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

// NextIDResponse contains the next available id.
type NextIDResponse struct {
	Data string `json:"data"`
}

func (p *ProxMox) getNextID() (string, error) {
	req, err := http.NewRequest("GET", p.apiURL+"/api2/json/cluster/nextid", nil)
	if err != nil {
		return "", err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("Authorization", "PVEAPIToken="+p.tokenID+"="+p.secret)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = p.CheckResultType(body, "getnextid", "")
	if err != nil {
		return "", err
	}

	ir := &NextIDResponse{}
	err = json.Unmarshal([]byte(body), ir)
	return ir.Data, nil
}

// CreateInstance - Creates instance on Proxmox.
func (p *ProxMox) CreateInstance(ctx *lepton.Context) error {

	var err error

	config := ctx.Config()

	nextid, err := p.getNextID()
	if err != nil {
		return err
	}

	p.instanceName = config.RunConfig.InstanceName

	p.isoStorageName = config.TargetConfig["IsoStorageName"]

	p.imageName = config.CloudConfig.ImageName

	p.arch = "x86_64"
	if config.TargetConfig["Arch"] != "" {
		p.arch = config.TargetConfig["Arch"]
	}

	p.machine = "q35"
	if config.TargetConfig["Machine"] != "" {
		p.machine = config.TargetConfig["Machine"]
	}

	p.sockets = "1"
	if config.TargetConfig["Sockets"] != "" {
		socketsInt, err := strconv.Atoi(config.TargetConfig["Sockets"])
		if err != nil {
			return err
		}
		if socketsInt < 1 {
			return errors.New("Bad configuration option; Sockets can only be set postitive starting from \"1\"")
		}
		p.sockets = config.TargetConfig["Sockets"]
	}

	p.cores = "1"
	if config.TargetConfig["Cores"] != "" {
		coresInt, err := strconv.Atoi(config.TargetConfig["Cores"])
		if err != nil {
			return err
		}
		if coresInt < 1 {
			return errors.New("Bad configuration option; Cores can only be set postitive starting from \"1\"")
		}
		p.cores = config.TargetConfig["Cores"]
	}

	p.numa = "0"
	if config.TargetConfig["Numa"] != "" {
		if config.TargetConfig["Numa"] != "0" && config.TargetConfig["Numa"] != "1" {
			return errors.New("Bad configuration option; Numa can only be set to \"0\" or \"1\"")
		}
		p.numa = config.TargetConfig["Numa"]
	}

	p.memory = "512"
	if config.TargetConfig["Memory"] != "" {
		memoryInt, err := lepton.RAMInBytes(config.TargetConfig["Memory"])
		if err != nil {
			return err
		}
		memoryInt = memoryInt / 1024 / 1024
		p.memory = strconv.FormatInt(memoryInt, 10)
	}

	p.storageName = "local-lvm"
	if config.TargetConfig["StorageName"] != "" {
		p.storageName = config.TargetConfig["StorageName"]
	}

	p.isoStorageName = "local"
	if config.TargetConfig["IsoStorageName"] != "" {
		p.isoStorageName = config.TargetConfig["IsoStorageName"]
	}

	p.bridgePrefix = "vmbr"
	if config.TargetConfig["BridgePrefix"] != "" {
		p.bridgePrefix = config.TargetConfig["BridgePrefix"]
	}

	p.onboot = "0"
	if config.TargetConfig["Onboot"] != "" {
		if config.TargetConfig["Onboot"] != "0" && config.TargetConfig["Onboot"] != "1" {
			return errors.New("Bad configuration option; Onboot can only be set to \"0\" or \"1\"")
		}
		p.onboot = config.TargetConfig["Onboot"]
	}

	p.protection = "0"
	if config.TargetConfig["Protection"] != "" {
		if config.TargetConfig["Protection"] != "0" && config.TargetConfig["Protection"] != "1" {
			return errors.New("Bad configuration option; Protection can only be set to \"0\" or \"1\"")
		}
		p.protection = config.TargetConfig["Protection"]
	}

	// These two preventive checks here, because Proxmox will not return
	// an error if the storage is missing and a misconfigured instance will be created.

	err = p.CheckStorage(p.storageName, "images")
	if err != nil {
		return err
	}

	err = p.CheckStorage(p.isoStorageName, "iso")
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("vmid", nextid)
	data.Set("name", p.instanceName)
	data.Set("name", p.imageName)
	data.Set("machine", p.machine)
	data.Set("sockets", p.sockets)
	data.Set("cores", p.cores)
	data.Set("numa", p.numa)
	data.Set("memory", p.memory)
	data.Set("onboot", p.onboot)
	data.Set("protection", p.protection)
	data.Set("serial0", "socket")

	// Configuring network interfaces

	nics := config.RunConfig.Nics
	for i := 0; i < len(nics); i++ {
		is := strconv.Itoa(i)
		brName := nics[i].BridgeName
		if brName == "" {
			brName = p.bridgePrefix + is
		}

		err = p.CheckBridge(brName)
		if err != nil {
			return err
		}

		if nics[i].IPAddress != "" {
			cidr := "24"

			if nics[i].NetMask != "" {
				cidrInt := lepton.CCidr(nics[i].NetMask)
				cidr = strconv.FormatInt(int64(cidrInt), 10)
			}

			if nics[i].Gateway != "" {
				data.Set("ipconfig"+is, "ip="+nics[i].IPAddress+"/"+cidr+","+"gw="+nics[i].Gateway)
			} else {
				data.Set("ipconfig"+is, "ip="+nics[i].IPAddress+"/"+cidr)
			}
		} else {
			data.Set("ipconfig"+is, "dhcp")
		}

		data.Set("net"+is, "model=virtio,bridge="+brName)
	}
	if len(nics) == 0 {
		// single dhcp nic
		data.Set("net0", "model=virtio,bridge=vmbr0")
	}

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu", bytes.NewBufferString(data.Encode()))
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

	err = p.CheckResultType(body, "createinstance", "file="+p.isoStorageName+":iso/"+p.imageName+".iso")
	if err != nil {
		return err
	}

	err = p.addVirtioDisk(ctx, nextid)
	if err != nil {
		return err
	}

	return err
}

func (p *ProxMox) movDisk(ctx *lepton.Context, vmid string) error {
	data := url.Values{}
	data.Set("disk", "virtio0")
	data.Set("node", p.nodeNAME)
	data.Set("format", "raw")
	data.Set("storage", p.isoStorageName)
	data.Set("vmid", vmid)

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/move_disk", bytes.NewBufferString(data.Encode()))
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

	return p.CheckResultType(body, "movdisk", p.storageName)
}

// Ticket is proxmox auth ticket.
type Ticket struct {
	Data TicketData `json:"data"`
}

// TicketData contains data for proxmox authn.
type TicketData struct {
	Ticket string `json:"ticket"`
	CSRF   string `json:"CSRFPreventionToken"`
}

// good for subsequent requests over 2? hrs
// you have to use this if you use absolue paths when setting disk
// perhaps there is a better work-around for that.
func (p *ProxMox) getTicket() (Ticket, error) {
	puser := os.Getenv("PROXMOX_USER")
	pass := os.Getenv("PROXMOX_PASS")

	t := Ticket{}

	if puser == "" || pass == "" {
		return t, errors.New("instance creation currently requires both PROXMOX_USER and PROXMOX_PASS to be set to root user or equivalent")
	}

	var data = strings.NewReader(`username=` + puser + "&password=" + pass)

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/access/ticket", data)
	if err != nil {
		return t, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal([]byte(body), &t)

	if t.Data.CSRF == "" || t.Data.Ticket == "" {
		return t, errors.New("empty ticket")
	}

	return t, err
}

func (p *ProxMox) addVirtioDisk(ctx *lepton.Context, vmid string) error {
	ticket, err := p.getTicket()
	if err != nil {
		return err
	}

	data := url.Values{}

	data.Set("virtio0", "local:0,import-from=/var/lib/vz/import/"+p.imageName+".raw")

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/config", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req.Header.Add("CSRFPreventionToken", ticket.Data.CSRF)
	clientCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: ticket.Data.Ticket,
	}
	req.AddCookie(clientCookie)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("can't add disk %s", string(body))
	}

	err = p.CheckResultType(body, "addvirtiodisk", p.isoStorageName)
	if err != nil {
		return err
	}

	// set boot order, needs to come after attaching disk
	data.Set("boot", "order=virtio0")

	req, err = http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+vmid+"/config", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("CSRFPreventionToken", ticket.Data.CSRF)
	clientCookie = &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: ticket.Data.Ticket,
	}
	req.AddCookie(clientCookie)

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("can't add disk %s", string(body))
	}

	return p.CheckResultType(body, "bootorderset", "")
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
	_, err = io.ReadAll(resp.Body)
	return err

}

// RebootInstance reboots the instance.
func (p *ProxMox) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance in Proxmox
func (p *ProxMox) StartInstance(ctx *lepton.Context, instanceID string) error {

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+instanceID+"/status/start", nil)
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
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// StopInstance halts instance from Proxmox.
func (p *ProxMox) StopInstance(ctx *lepton.Context, instanceID string) error {

	req, err := http.NewRequest("POST", p.apiURL+"/api2/json/nodes/"+p.nodeNAME+"/qemu/"+instanceID+"/status/stop", nil)
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
	_, err = io.ReadAll(resp.Body)
	if err != nil {
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

// InstanceStats show metrics for instances on proxmox
func (p *ProxMox) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not avilable")
}
