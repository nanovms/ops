//go:build relayered || !onlyprovider

package relayered

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance - Creates instance on relayered Cloud Platform
func (v *relayered) CreateInstance(ctx *lepton.Context) error {

	uri := "http://dev.relayered.net/instances/create"

	stuff := `{}`

	reqBody := []byte(stuff)

	client := &http.Client{}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("RELAYERED_TOKEN", v.token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(body))
	}

	fmt.Println(string(body))

	return nil
}

// GetInstanceByName returns instance with given name
func (v *relayered) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// InstancesListResponse is the set of instances available from relayered in an
// images list call.
type InstancesListResponse []Instance

// Instance represents a virtual server instance.
type Instance struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Created   string `json:"created"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetInstances return all instances on relayered
func (v *relayered) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {

	uri := "http://dev.relayered.net/instances/list"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("RELAYERED_TOKEN", v.token)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(body))

	var ilr InstancesListResponse
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	var cloudInstances []lepton.CloudInstance

	for _, instance := range ilr {
		cloudInstances = append(cloudInstances, lepton.CloudInstance{
			ID:        instance.ID,
			Status:    instance.Status,
			Created:   instance.CreatedAt,
			PublicIps: []string{""},
			Image:     "",
		})
	}

	return cloudInstances, nil

}

// ListInstances lists instances on v
func (v *relayered) ListInstances(ctx *lepton.Context) error {
	instances, err := v.GetInstances(ctx)
	if err != nil {
		fmt.Println(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "MainIP", "Status", "ImageID"})
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

// DeleteInstance deletes instance from relayered
func (v *relayered) DeleteInstance(ctx *lepton.Context, instanceID string) error {

	uri := "http://dev.relayered.net/instances/delete/" + instanceID

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("RELAYERED_TOKEN", v.token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(body))

	return nil
}

// StartInstance starts an instance in relayered
func (v *relayered) StartInstance(ctx *lepton.Context, instanceID string) error {
	return nil
}

// StopInstance halts instance from v
func (v *relayered) StopInstance(ctx *lepton.Context, instanceID string) error {
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *relayered) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (v *relayered) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}
