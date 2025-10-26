package kamatera

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance launches a server in Kamatera Cloud using the configured snapshot.
func (h *Kamatera) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	if c.CloudConfig.Flavor == "" {
		c.CloudConfig.Flavor = "DEV1-S"
	}

	_, err := h.getImageByName(ctx, c.CloudConfig.ImageName)
	if err != nil {
		fmt.Println(err)
	}

	//	instanceName := c.RunConfig.InstanceName

	if err != nil {
		panic(err)
	}

	return err
}

// ListInstances prints all managed Kamatera instances in table or JSON form.
func (h *Kamatera) ListInstances(ctx *lepton.Context) error {
	instances, err := h.GetInstances(ctx)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(instances)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID", "Status", "Created", "Private IPs", "Public IPs", "Image"})
	table.SetRowLine(true)

	for _, instance := range instances {
		table.Append([]string{
			instance.Name,
			instance.ID,
			instance.Status,
			instance.Created,
			strings.Join(instance.PrivateIps, ","),
			strings.Join(instance.PublicIps, ","),
			instance.Image,
		})
	}

	table.Render()
	return nil
}

// InstanceStats returns an error because Kamatera metrics are not yet implemented.
func (h *Kamatera) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not available")
}

// GetInstances retrieves all instances managed by Ops on Kamatera Cloud.
func (h *Kamatera) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	//	c := ctx.Config()

	instances := []lepton.CloudInstance{}

	url := "https://console.kamatera.com/service/servers"
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return instances, err
	}
	fmt.Println(h.apiKey)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+h.apiKey)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return instances, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return instances, err
	}

	fmt.Println(string(body))

	/*for _, server := range response.Servers {
		pubips := []string{}
		for i := 0; i < len(server.PublicIPs); i++ {
			pubips = append(pubips, (*server.PublicIPs[i]).Address.String())
		}

		pip := ""
		if (*server).PrivateIP != nil {
			pip = *server.PrivateIP
		}

		instances = append(instances, lepton.CloudInstance{
			ID:         server.ID,
			Name:       server.Name,
			PublicIps:  pubips,
			PrivateIps: []string{pip},
			Status:     (*server).State.String(),
			Image:      (*server).Image.Name,
			Created:    (*server.CreationDate).String(),
		})

	}
	*/

	return instances, err
}

// GetInstanceByName looks up a managed Kamatera instance by its name label.
func (h *Kamatera) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	instances, err := h.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(instances); i++ {
		if instances[i].Name == name {
			return &instances[i], nil
		}
	}

	return nil, errors.New("can't find instance")
}

// DeleteInstance removes the specified Kamatera server.
func (h *Kamatera) DeleteInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	//	c := ctx.Config()
	return nil
}

// StopInstance powers off the target Kamatera server.
func (h *Kamatera) StopInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	//	c := ctx.Config()
	return nil
}

// StartInstance powers on the target Kamatera server.
func (h *Kamatera) StartInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	return nil
}

// RebootInstance restarts the target Kamatera server.
func (h *Kamatera) RebootInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")

	return nil
}

// GetInstanceLogs returns an error because Kamatera log streaming is not implemented.
func (*Kamatera) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", fmt.Errorf("GetInstanceLogs not yet implemented")
}

// PrintInstanceLogs returns an error because Kamatera log streaming is not implemented.
func (*Kamatera) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return fmt.Errorf("PrintInstanceLogs not yet implemented")
}
