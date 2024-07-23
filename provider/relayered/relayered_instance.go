//go:build relayered || !onlyprovider

package relayered

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

var baseURI = "http://dev.relayered.net"

// CreateInstance - Creates instance on relayered Cloud Platform
func (v *Relayered) CreateInstance(ctx *lepton.Context) error {

	uri := baseURI + "/instances/create"

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
func (v *Relayered) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
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
func (v *Relayered) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {

	uri := baseURI + "/instances/list"

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
func (v *Relayered) ListInstances(ctx *lepton.Context) error {
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
func (v *Relayered) DeleteInstance(ctx *lepton.Context, instanceID string) error {

	uri := baseURI + "/instances/delete/" + instanceID

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

// RebootInstance reboots the instance.
func (v *Relayered) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance in relayered
func (v *Relayered) StartInstance(ctx *lepton.Context, instanceID string) error {
	return nil
}

// StopInstance halts instance from v
func (v *Relayered) StopInstance(ctx *lepton.Context, instanceID string) error {
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *Relayered) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	s, err := v.GetInstanceLogs(ctx, instancename)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(s)

	return nil
}

// GetInstanceLogs gets instance related logs
func (v *Relayered) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	uri := baseURI + "/instances/logs/" + instancename

	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
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

	return string(body), nil
}

// InstanceStats show metrics for instances on relayered
func (p *Relayered) InstanceStats(ctx *lepton.Context) error {
	return errors.New("currently not avilable")
}
