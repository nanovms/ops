package linode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nanovms/ops/lepton"

	"github.com/olekukonko/tablewriter"
)

type createInstanceResponse struct {
	ID int `json:"id"`
}

type linodeStatusResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type diskStatusResponse struct {
	Data []DiskStatus `json:"data"`
}

// DiskStatus holds the status for a linode disk status.
type DiskStatus struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

// InstanceListResponse is the set of instances available from linode.
type InstanceListResponse struct {
	Data []Instance `json:"data"`
}

// Instance is the Linode configuration for a given instance.
type Instance struct {
	Created string   `json:"created"`
	ID      int      `json:"id"`
	Image   string   `json:"image"`
	IPv4    []string `json:"ipv4"`
	Region  string   `json:"region"`
	Status  string   `json:"status"`
}

func (v *Linode) createInstance(imgName string) int {
	client := &http.Client{}

	s := `{
	         "backups_enabled": false,
	         "swap_size": 512,
	         "root_pass": "aComplexP@ssword",
	         "booted": false,
	         "label": "` + imgName + `",
	         "type": "g6-nanode-1",
	         "region": "us-west",
	         "group": "Linode-Group"
	       }`

	reqBody := []byte(s)

	uri := "https://api.linode.com/v4/linode/instances"

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	// handle me better
	if res.StatusCode != 200 {
		sbody := string(body)
		fmt.Println(sbody)
	}

	cir := &createInstanceResponse{}
	err = json.Unmarshal(body, &cir)
	if err != nil {
		fmt.Println(err)
	}

	return cir.ID
}

func (v *Linode) addDisk(instanceID int, imageID int, imgName string) int {
	client := &http.Client{}

	simageID := strconv.Itoa(imageID)
	sinstanceID := strconv.Itoa(instanceID)

	s := `{
	         "label": "` + imgName + `",
	         "image": "private/` + simageID + `",
	         "size": 1300,
	         "root_pass": "aComplexP@ssword"
	       }`

	reqBody := []byte(s)

	uri := "https://api.linode.com/v4/linode/instances/" + sinstanceID + "/disks"

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	// handle me better
	if res.StatusCode != 200 {
		sbody := string(body)
		fmt.Println(sbody)
	}

	ds := &DiskStatus{}
	err = json.Unmarshal(body, &ds)
	if err != nil {
		fmt.Println(err)
	}

	return ds.ID
}

func (v *Linode) addConfig(instanceID int, diskID int, imgName string) {
	client := &http.Client{}

	sinstanceID := strconv.Itoa(instanceID)
	sdiskID := strconv.Itoa(diskID)

	s := `{
	         "kernel": "linode/direct-disk",
	         "comments": "This is my main Config",
	         "memory_limit": 1024,
	         "run_level": "default",
	         "virt_mode": "paravirt",
	         "helpers": {
	           "updatedb_disabled": false,
	           "distro": false,
	           "modules_dep": false,
	           "network": false,
	           "devtmpfs_automount": false
	         },
	         "label": "` + imgName + `",
	         "devices": {
	           "sda": {
	             "disk_id": ` + sdiskID + `,
	             "volume_id": null
	           }
	         }
	       }`

	reqBody := []byte(s)

	uri := "https://api.linode.com/v4/linode/instances/" + sinstanceID + "/configs"

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

}

// CreateInstance - Creates instance on Digital Ocean Platform
func (v *Linode) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	t := time.Now().Unix()
	st := strconv.FormatInt(t, 10)

	imgName := c.CloudConfig.ImageName + "-" + st

	fmt.Printf("creating instance %s\n", imgName)

	instanceID := v.createInstance(imgName)

	status := ""
	for {
		status = v.getStatusForLinode(instanceID)
		time.Sleep(3 * time.Second) //hack

		if status == "offline" {
			break
		}
	}

	imgs, err := v.GetImages(ctx)
	if err != nil {
		fmt.Println(err)
	}

	imgID := 0

	for i := 0; i < len(imgs); i++ {

		if imgs[i].Name == c.CloudConfig.ImageName {
			s := strings.Split(imgs[i].ID, "private/")
			imgID, err = strconv.Atoi(s[1])
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	diskID := v.addDisk(instanceID, imgID, imgName)

	status = ""
	for {
		status = v.getStatusForDisk(instanceID)
		time.Sleep(3 * time.Second) // hack

		if status == "ready" {
			break
		}
	}

	v.addConfig(instanceID, diskID, imgName)

	sinstanceID := strconv.Itoa(instanceID)

	v.StartInstance(ctx, sinstanceID)

	return nil
}

func (v *Linode) getStatusForDisk(id int) string {
	client := &http.Client{}

	sinstanceID := strconv.Itoa(id)

	uri := "https://api.linode.com/v4/linode/instances/" + sinstanceID + "/disks"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	dsr := &diskStatusResponse{}
	err = json.Unmarshal(body, &dsr)
	if err != nil {
		fmt.Println(err)
	}

	// seeems like some transient empties
	if len(dsr.Data) > 0 {
		return dsr.Data[0].Status
	}

	return ""
}

func (v *Linode) getStatusForLinode(id int) string {
	client := &http.Client{}

	sinstanceID := strconv.Itoa(id)

	uri := "https://api.linode.com/v4/linode/instances/" + sinstanceID

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	lsr := &linodeStatusResponse{}
	err = json.Unmarshal(body, &lsr)
	if err != nil {
		fmt.Println(err)
	}

	return lsr.Status
}

// GetInstanceByName returns instance with given name
func (v *Linode) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstances return all instances on Linode
func (v *Linode) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	client := &http.Client{}

	uri := "https://api.linode.com/v4/linode/instances"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	ilr := &InstanceListResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	var cloudInstances []lepton.CloudInstance

	for _, instance := range ilr.Data {
		cloudInstances = append(cloudInstances, lepton.CloudInstance{
			ID:        strconv.Itoa(instance.ID),
			Status:    instance.Status,
			Created:   instance.Created,
			PublicIps: instance.IPv4,
			Image:     instance.Image, // made from disk not image so prob need to get from disk which is not actually in this call..
		})
	}

	return cloudInstances, nil

}

// ListInstances lists instances on v
func (v *Linode) ListInstances(ctx *lepton.Context) error {
	instances, err := v.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "MainIP", "Status", "ImageID"}) // "Region"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, instance := range instances {
		var row []string
		row = append(row, instance.ID)
		row = append(row, instance.PublicIps[0])
		row = append(row, instance.Status)
		row = append(row, instance.Image) /// Os)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from linode
func (v *Linode) DeleteInstance(ctx *lepton.Context, instanceID string) error {
	client := &http.Client{}

	uri := "https://api.linode.com/v4/linode/instances/" + instanceID

	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// StartInstance starts an instance in linode
func (v *Linode) StartInstance(ctx *lepton.Context, instanceID string) error {
	client := &http.Client{}

	uri := "https://api.linode.com/v4/linode/instances/" + instanceID + "/boot"

	req, err := http.NewRequest("POST", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	_, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

// StopInstance halts instance from v
func (v *Linode) StopInstance(ctx *lepton.Context, instanceID string) error {
	// POST https://api.linode.com/v4/linode/instances/{linodeId}/shutdown
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *Linode) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Linode) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}
