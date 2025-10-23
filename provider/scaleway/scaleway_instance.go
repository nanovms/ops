package scaleway

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance launches a server in Scaleway Cloud using the configured snapshot.
func (h *Scaleway) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	instanceAPI := instance.NewAPI(h.client)

	if c.CloudConfig.Flavor == "" {
		c.CloudConfig.Flavor = "DEV1-S"
	}

	i, err := h.getImageByName(ctx, c.CloudConfig.ImageName)
	if err != nil {
		fmt.Println(err)
	}

	instanceName := c.RunConfig.InstanceName

	createRes, err := instanceAPI.CreateServer(&instance.CreateServerRequest{
		Name:              instanceName,
		CommercialType:    c.CloudConfig.Flavor,
		Image:             scw.StringPtr(i.ID),
		DynamicIPRequired: scw.BoolPtr(true),
		Project:           scw.StringPtr(os.Getenv("SCALEWAY_ORGANIZATION_ID")),
	})
	if err != nil {
		panic(err)
	}

	timeout := 5 * time.Minute
	err = instanceAPI.ServerActionAndWait(&instance.ServerActionAndWaitRequest{
		ServerID: createRes.Server.ID,
		Action:   instance.ServerActionPoweron,
		Timeout:  &timeout,
	})
	if err != nil {
		panic(err)
	}
	return err
}

// ListInstances prints all managed Scaleway instances in table or JSON form.
func (h *Scaleway) ListInstances(ctx *lepton.Context) error {
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

// InstanceStats returns an error because Scaleway metrics are not yet implemented.
func (h *Scaleway) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not available")
}

// GetInstances retrieves all instances managed by Ops on Scaleway Cloud.
func (h *Scaleway) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	c := ctx.Config()

	instances := []lepton.CloudInstance{}

	instanceApi := instance.NewAPI(h.client)

	response, err := instanceApi.ListServers(&instance.ListServersRequest{
		Zone: scw.Zone(c.CloudConfig.Zone),
	})
	if err != nil {
		return instances, err
	}

	for _, server := range response.Servers {
		fmt.Println("%v", server)

		pubips := []string{}
		for i := 0; i < len(server.PublicIPs); i++ {
			pubips = append(pubips, (*server.PublicIPs[i]).Address.String())
		}

		instances = append(instances, lepton.CloudInstance{
			ID:         server.ID,
			Name:       server.Name,
			PublicIps:  pubips,
			PrivateIps: []string{*server.PrivateIP},
			Status:     server.State.String(),
			Image:      server.Image.Name,
			Created:    (*server.CreationDate).String(),
		})

	}

	return instances, err
}

// GetInstanceByName looks up a managed Scaleway instance by its name label.
func (h *Scaleway) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
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

// DeleteInstance removes the specified Scaleway server.
func (h *Scaleway) DeleteInstance(ctx *lepton.Context, instancename string) error {
	c := ctx.Config()

	instanceAPI := instance.NewAPI(h.client)

	i, err := h.GetInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	return instanceAPI.DeleteServer(&instance.DeleteServerRequest{
		Zone:     scw.Zone(c.CloudConfig.Zone),
		ServerID: i.ID,
	})
}

// StopInstance powers off the target Scaleway server.
func (h *Scaleway) StopInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	return nil
}

// StartInstance powers on the target Scaleway server.
func (h *Scaleway) StartInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	return nil
}

// RebootInstance restarts the target Scaleway server.
func (h *Scaleway) RebootInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")

	return nil
}

// GetInstanceLogs returns an error because Scaleway log streaming is not implemented.
func (*Scaleway) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", fmt.Errorf("GetInstanceLogs not yet implemented")
}

// PrintInstanceLogs returns an error because Scaleway log streaming is not implemented.
func (*Scaleway) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return fmt.Errorf("PrintInstanceLogs not yet implemented")
}
