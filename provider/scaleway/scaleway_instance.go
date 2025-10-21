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

	client, err := scw.NewClient(
		scw.WithAuth("SCALEWAY_ACCESS_KEY_ID", "SCALEWAY_SECRET_ACCESS_KEY"),
		scw.WithDefaultOrganizationID("SCALEWAY_ORGANIZATION_ID"),
		scw.WithDefaultZone(scw.ZoneFrPar1),
	)
	if err != nil {
		panic(err)
	}

	instanceAPI := instance.NewAPI(client)

	serverType := "DEV1-S"
	image := "ubuntu_focal"

	createRes, err := instanceAPI.CreateServer(&instance.CreateServerRequest{
		Name:              "my-server-01",
		CommercialType:    serverType,
		Image:             scw.StringPtr(image),
		DynamicIPRequired: scw.BoolPtr(true),
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

	client, err := scw.NewClient(
		scw.WithDefaultOrganizationID("SCALEWAY_ORGANIZATION_ID"),
		scw.WithAuth("SCALEWAY_ACCESS_KEY_ID", "SCALEWAY_SECRET_ACCESS_KEY"),
		scw.WithDefaultRegion("SCW_REGION"),
	)
	if err != nil {
		panic(err)
	}

	instanceApi := instance.NewAPI(client)

	response, err := instanceApi.ListServers(&instance.ListServersRequest{
		Zone: scw.ZoneFrPar1,
	})
	if err != nil {
		panic(err)
	}

	for _, server := range response.Servers {
		fmt.Println("Server", server.ID, server.Name)
		//		result = append(result, toCloudInstance(server))
	}

	return nil, nil
}

// GetInstanceByName looks up a managed Scaleway instance by its name label.
func (h *Scaleway) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	log.Warn("not yet implemented")
	return nil, nil
}

// DeleteInstance removes the specified Scaleway server.
func (h *Scaleway) DeleteInstance(ctx *lepton.Context, instancename string) error {
	log.Warn("not yet implemented")
	return nil
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
