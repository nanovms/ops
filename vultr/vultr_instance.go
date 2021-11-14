package vultr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance - Creates instance on Digital Ocean Platform
func (v *Vultr) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	// you may poll /v1/server/list?SUBID=<SUBID> and check that the "status" field is set to "active"

	createURL := "https://api.vultr.com/v1/server/create"

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("DCID", "1")

	// this is the instance size
	// TODO
	urlData.Set("VPSPLANID", "201")

	// id for snapshot
	urlData.Set("OSID", "164")
	urlData.Set("SNAPSHOTID", c.CloudConfig.ImageName)

	req, err := http.NewRequest("POST", createURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))

	return nil
}

// GetInstanceByName returns instance with given name
func (v *Vultr) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// GetInstances return all instances on Vultr
// TODO
func (v *Vultr) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// ListInstances lists instances on v
func (v *Vultr) ListInstances(ctx *lepton.Context) error {

	client := http.Client{}
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/server/list", nil)
	if err != nil {
		log.Fatal(err)
	}
	token := os.Getenv("VULTR_TOKEN")

	req.Header.Set("API-Key", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data map[string]vultrServer

	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Error(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)

	for _, image := range data {
		var row []string
		row = append(row, image.Name)
		row = append(row, image.Status)
		row = append(row, image.CreatedAt)
		row = append(row, image.PrivateIP)
		row = append(row, image.PublicIP)
		table.Append(row)
	}

	table.Render()

	return nil
}

// DeleteInstance deletes instance from v
func (v *Vultr) DeleteInstance(ctx *lepton.Context, instanceID string) error {
	destroyInstanceURL := "https://api.vultr.com/v1/server/destroy"

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("SUBID", instanceID)

	req, err := http.NewRequest("POST", destroyInstanceURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))
	return nil
}

// StartInstance starts an instance in v
func (v *Vultr) StartInstance(ctx *lepton.Context, instanceID string) error {
	startInstanceURL := "https://api.vultr.com/v1/server/start"

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("SUBID", instanceID)

	req, err := http.NewRequest("POST", startInstanceURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))
	return nil
}

// StopInstance halts instance from v
func (v *Vultr) StopInstance(ctx *lepton.Context, instanceID string) error {
	haltInstanceURL := "https://api.vultr.com/v1/server/halt"

	token := os.Getenv("VULTR_TOKEN")

	urlData := url.Values{}
	urlData.Set("SUBID", instanceID)

	req, err := http.NewRequest("POST", haltInstanceURL, strings.NewReader(urlData.Encode()))
	req.Header.Set("API-Key", token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info("response Body:", string(body))
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *Vultr) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := v.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Vultr) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}
