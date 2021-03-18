package upcloud

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
	"github.com/olekukonko/tablewriter"
)

var (
	opsTag = upcloud.Tag{
		Name:        "OPS",
		Description: "Created by ops",
	}
)

// CreateInstance uses a template to launch a server in upcloud
func (p *Provider) CreateInstance(ctx *lepton.Context) error {

	image, err := p.getImageByName(ctx, ctx.Config().CloudConfig.ImageName)
	if err != nil {
		return err
	}

	instanceName := ctx.Config().RunConfig.InstanceName

	storageDevices := request.CreateServerStorageDeviceSlice{
		{
			Title:   instanceName,
			Action:  "clone",
			Storage: image.ID,
		},
	}

	createInstanceReq := &request.CreateServerRequest{
		Hostname:       instanceName,
		Title:          instanceName,
		StorageDevices: storageDevices,
		Zone:           p.zone,
	}

	ctx.Logger().Info("creating server")

	serverDetails, err := p.upcloud.CreateServer(createInstanceReq)
	if err != nil {
		return err
	}

	ctx.Logger().Debug("%+v", serverDetails)

	ctx.Logger().Info("getting ops tags")
	opsTag, err := p.findOrCreateTag(opsTag)
	if err != nil {
		ctx.Logger().Warn("failed creating ops tag: %s", err)
		return nil
	}

	imageTag, err := p.findOrCreateTag(upcloud.Tag{
		Name:        "image-" + image.Name,
		Description: "Creted with image " + image.Name,
	})
	if err != nil {
		ctx.Logger().Warn("failed creating image tag: %s", err)
		return nil
	}

	ctx.Logger().Info("assigning ops tags")

	assignOpsTagsRequest := &request.TagServerRequest{
		UUID: serverDetails.UUID,
		Tags: []string{opsTag.Name, imageTag.Name},
	}

	_, err = p.upcloud.TagServer(assignOpsTagsRequest)
	if err != nil {
		ctx.Logger().Warn("failed assigning ops tags: %s", err)
		return nil
	}

	return nil
}

// ListInstances prints servers list managed by upcloud in table
func (p *Provider) ListInstances(ctx *lepton.Context) (err error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Status", "Private Ips", "Public Ips", "Image"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

	table.SetRowLine(true)

	for _, i := range instances {
		var rows []string

		rows = append(rows, i.ID)
		rows = append(rows, i.Name)
		rows = append(rows, i.Status)
		rows = append(rows, strings.Join(i.PrivateIps, ", "))
		rows = append(rows, strings.Join(i.PublicIps, ", "))
		rows = append(rows, i.Image)

		table.Append(rows)
	}

	table.Render()

	return
}

// GetInstances returns the list of servers managed by upcloud
func (p *Provider) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	instances = []lepton.CloudInstance{}
	serversIDs := []string{}

	opsTag, err := p.findOrCreateTag(opsTag)
	if err != nil {
		ctx.Logger().Warn("failed creating tags: %s", err)

		var servers *upcloud.Servers
		servers, err = p.upcloud.GetServers()
		if err != nil {
			return
		}

		for _, s := range servers.Servers {
			serversIDs = append(serversIDs, s.UUID)
		}
	} else {
		for _, s := range opsTag.Servers {
			serversIDs = append(serversIDs, s)
		}
	}

	instancesCh := make(chan *lepton.CloudInstance)
	errCh := make(chan error)

	for _, id := range serversIDs {
		go p.asyncGetInstance(ctx, id, instancesCh, errCh)
	}

	for i := 0; i < len(serversIDs); i++ {
		select {
		case instance := <-instancesCh:
			instances = append(instances, *instance)
		case err = <-errCh:
			return
		}
	}

	return
}

func (p *Provider) asyncGetInstance(ctx *lepton.Context, id string, instancesCh chan *lepton.CloudInstance, errCh chan error) {
	var instance *lepton.CloudInstance

	instance, err := p.GetInstanceByID(ctx, id)
	if err != nil {
		errCh <- err
		return
	}

	instancesCh <- instance
}

// DeleteInstance removes server
func (p *Provider) DeleteInstance(ctx *lepton.Context, instancename string) (err error) {
	instance, err := p.getInstanceByName(ctx, instancename)
	if err != nil {
		return
	}

	if instance.Status != "stopped" {
		err = p.stopServer(instance.ID)
		if err != nil {
			ctx.Logger().Warn("failed stopping server: %s", err)
		}

		err = p.waitForServerState(instance.ID, "stopped")
		if err != nil {
			return
		}
	}

	deleteServerReq := &request.DeleteServerRequest{
		UUID: instance.ID,
	}

	ctx.Logger().Debug(`deleting server with uuid "%s"`, instance.ID)
	err = p.upcloud.DeleteServer(deleteServerReq)

	return
}

// StopInstance stops server in upcloud
func (p *Provider) StopInstance(ctx *lepton.Context, instancename string) (err error) {
	instance, err := p.getInstanceByName(ctx, instancename)
	if err != nil {
		return
	}

	ctx.Logger().Debug(`stopping server with uuid "%s"`, instance.ID)
	err = p.stopServer(instance.ID)

	return
}

func (p *Provider) stopServer(uuid string) (err error) {
	stopServerReq := &request.StopServerRequest{
		UUID: uuid,
	}

	_, err = p.upcloud.StopServer(stopServerReq)

	return
}

// StartInstance initiates server in upcloud
func (p *Provider) StartInstance(ctx *lepton.Context, instancename string) (err error) {
	instance, err := p.getInstanceByName(ctx, instancename)
	if err != nil {
		return
	}

	ctx.Logger().Debug(`starting server with uuid "%s"`, instance.ID)

	err = p.startServer(instance.ID)

	return
}

func (p *Provider) startServer(uuid string) (err error) {
	startServerReq := &request.StartServerRequest{
		UUID: uuid,
	}

	_, err = p.upcloud.StartServer(startServerReq)

	return
}

func (p *Provider) getInstanceByName(ctx *lepton.Context, name string) (instance *lepton.CloudInstance, err error) {
	ctx.Logger().Debug(`getting instance by name "%s"`, name)
	server, err := p.getServerByName(ctx, name)
	if err != nil {
		return
	}

	instance, err = p.GetInstanceByID(ctx, server.UUID)

	return
}

func (p *Provider) getServerByName(ctx *lepton.Context, name string) (server *upcloud.Server, err error) {
	servers, err := p.upcloud.GetServers()
	if err != nil {
		return
	}

	for _, s := range servers.Servers {
		if s.Title == name {

			server = &s

			return
		}
	}

	err = fmt.Errorf(`server with title "%s" not found`, name)

	return
}

// GetInstanceByID return a upcloud server details with ID specified
func (p *Provider) GetInstanceByID(ctx *lepton.Context, id string) (instance *lepton.CloudInstance, err error) {
	var serverDetails *upcloud.ServerDetails

	serverDetailsReq := &request.GetServerDetailsRequest{UUID: id}

	serverDetails, err = p.upcloud.GetServerDetails(serverDetailsReq)
	if err != nil {
		return
	}

	publicIPS := []string{}
	privateIPS := []string{}

	for _, ip := range serverDetails.IPAddresses {
		if ip.Access == "public" {
			publicIPS = append(publicIPS, ip.Address)
		} else {
			privateIPS = append(privateIPS, ip.Address)
		}
	}

	imageName := ""
	for _, t := range serverDetails.Tags {
		if strings.Contains(t, "image-") {
			parts := strings.Split(t, "-")
			if len(parts) > 1 {
				imageName = strings.Join(parts[1:], "-")
			}
		}
	}

	instance = &lepton.CloudInstance{
		ID:         serverDetails.UUID,
		Name:       serverDetails.Title,
		Status:     serverDetails.State,
		PublicIps:  publicIPS,
		PrivateIps: privateIPS,
		Image:      imageName,
	}

	return
}

// GetInstanceLogs is a stub
func (p *Provider) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", errors.New("Unsupported")
}

// PrintInstanceLogs prints server log on console
func (p *Provider) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("Unsupported")
}

func (p *Provider) waitForServerState(uuid, state string) (err error) {

	waitReq := &request.WaitForServerStateRequest{
		UUID:         uuid,
		DesiredState: state,
		Timeout:      1 * time.Minute,
	}

	_, err = p.upcloud.WaitForServerState(waitReq)

	return
}
