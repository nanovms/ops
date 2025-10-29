package hetzner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance launches a server in Hetzner Cloud using the configured snapshot.
func (h *Hetzner) CreateInstance(ctx *lepton.Context) error {
	config := ctx.Config()

	image, err := h.fetchSnapshotByName(context.Background(), config.CloudConfig.ImageName)
	if err != nil {
		return err
	}
	if image == nil {
		return fmt.Errorf(`image with name "%s" not found`, config.CloudConfig.ImageName)
	}

	flavor := strings.TrimSpace(config.CloudConfig.Flavor)
	if flavor == "" {
		flavor = defaultServerType
	}

	serverType, _, err := h.Client.ServerType.GetByName(context.Background(), flavor)
	if err != nil {
		return err
	}
	if serverType == nil {
		return fmt.Errorf("server type %q not found", flavor)
	}

	instanceName := strings.TrimSpace(config.RunConfig.InstanceName)
	if instanceName == "" {
		instanceName = fmt.Sprintf("%s-%d", config.CloudConfig.ImageName, time.Now().Unix())
	}

	zone := strings.TrimSpace(config.CloudConfig.Zone)

	_, err = h.createServer(context.Background(), serverCreateParams{
		Name:       instanceName,
		ServerType: serverType,
		Image:      image,
		BaseLabels: map[string]string{
			opsInstanceLabelKey:  config.CloudConfig.ImageName,
			opsImageNameLabelKey: config.CloudConfig.ImageName,
		},
		Tags:         config.CloudConfig.Tags,
		TagFilter:    func(tag types.Tag) bool { return tag.IsInstanceLabel() },
		LocationName: zone,
		UserData:     config.CloudConfig.UserData,
	})
	return err
}

// ListInstances prints all managed Hetzner instances in table or JSON form.
func (h *Hetzner) ListInstances(ctx *lepton.Context) error {
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

// InstanceStats returns an error because Hetzner metrics are not yet implemented.
func (h *Hetzner) InstanceStats(ctx *lepton.Context, instancename string, watch bool) error {
	return errors.New("currently not available")
}

// GetInstances retrieves all instances managed by Ops on Hetzner Cloud.
func (h *Hetzner) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: managedLabelSelector(),
		},
	}

	servers, err := h.Client.Server.AllWithOpts(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	result := make([]lepton.CloudInstance, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		if labelValue(server.Labels, opsLabelKey) != opsLabelValue {
			continue
		}
		result = append(result, toCloudInstance(server))
	}

	return result, nil
}

// GetInstanceByName looks up a managed Hetzner instance by its name label.
func (h *Hetzner) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	server, err := h.fetchServerByName(context.Background(), name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, lepton.ErrInstanceNotFound(name)
	}

	instance := toCloudInstance(server)
	return &instance, nil
}

// DeleteInstance removes the specified Hetzner server.
func (h *Hetzner) DeleteInstance(ctx *lepton.Context, instancename string) error {
	server, err := h.fetchServerByName(context.Background(), instancename)
	if err != nil {
		return err
	}
	if server == nil {
		return lepton.ErrInstanceNotFound(instancename)
	}

	if _, _, err := h.Client.Server.DeleteWithResult(context.Background(), server); err != nil {
		return err
	}

	return nil
}

// StopInstance powers off the target Hetzner server.
func (h *Hetzner) StopInstance(ctx *lepton.Context, instancename string) error {
	server, err := h.fetchServerByName(context.Background(), instancename)
	if err != nil {
		return err
	}
	if server == nil {
		return lepton.ErrInstanceNotFound(instancename)
	}

	action, _, err := h.Client.Server.Poweroff(context.Background(), server)
	if err != nil {
		return err
	}
	if action != nil {
		return h.Client.Action.WaitFor(context.Background(), action)
	}
	return nil
}

// StartInstance powers on the target Hetzner server.
func (h *Hetzner) StartInstance(ctx *lepton.Context, instancename string) error {
	server, err := h.fetchServerByName(context.Background(), instancename)
	if err != nil {
		return err
	}
	if server == nil {
		return lepton.ErrInstanceNotFound(instancename)
	}

	action, _, err := h.Client.Server.Poweron(context.Background(), server)
	if err != nil {
		return err
	}
	if action != nil {
		return h.Client.Action.WaitFor(context.Background(), action)
	}
	return nil
}

// RebootInstance restarts the target Hetzner server.
func (h *Hetzner) RebootInstance(ctx *lepton.Context, instancename string) error {
	server, err := h.fetchServerByName(context.Background(), instancename)
	if err != nil {
		return err
	}
	if server == nil {
		return lepton.ErrInstanceNotFound(instancename)
	}

	action, _, err := h.Client.Server.Reboot(context.Background(), server)
	if err != nil {
		return err
	}
	if action != nil {
		return h.Client.Action.WaitFor(context.Background(), action)
	}
	return nil
}

// GetInstanceLogs returns an error because Hetzner log streaming is not implemented.
func (*Hetzner) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", fmt.Errorf("GetInstanceLogs not yet implemented")
}

// PrintInstanceLogs returns an error because Hetzner log streaming is not implemented.
func (*Hetzner) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return fmt.Errorf("PrintInstanceLogs not yet implemented")
}

func (h *Hetzner) fetchServerByName(ctx context.Context, name string) (*hcloud.Server, error) {
	server, _, err := h.Client.Server.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}
	if labelValue(server.Labels, opsLabelKey) != opsLabelValue {
		return nil, nil
	}
	return server, nil
}

type serverCreateParams struct {
	Name         string
	ServerType   *hcloud.ServerType
	Image        *hcloud.Image
	UserData     string
	SSHKeys      []*hcloud.SSHKey
	BaseLabels   map[string]string
	Tags         []types.Tag
	TagFilter    func(types.Tag) bool
	LocationName string
}

func (h *Hetzner) createServer(ctx context.Context, params serverCreateParams) (*hcloud.Server, error) {
	filter := params.TagFilter
	if filter == nil {
		filter = func(tag types.Tag) bool { return true }
	}

	labels := combineLabels(params.BaseLabels, params.Tags, filter)
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, ok := labels[opsLabelKey]; !ok {
		labels[opsLabelKey] = opsLabelValue
	}

	opts := hcloud.ServerCreateOpts{
		Name:       params.Name,
		ServerType: params.ServerType,
		Image:      params.Image,
		UserData:   params.UserData,
		SSHKeys:    params.SSHKeys,
		Labels:     labels,
	}

	if params.LocationName != "" {
		opts.Location = &hcloud.Location{Name: params.LocationName}
	}

	result, _, err := h.Client.Server.Create(ctx, opts)
	if err != nil {
		return nil, err
	}

	if result.Action != nil {
		if err := h.Client.Action.WaitFor(ctx, result.Action); err != nil {
			return nil, err
		}
	}

	if result.Server == nil {
		return nil, fmt.Errorf("hetzner api returned empty server result")
	}

	return result.Server, nil
}

func toCloudInstance(server *hcloud.Server) lepton.CloudInstance {
	privateIPs := make([]string, 0, len(server.PrivateNet))
	for _, iface := range server.PrivateNet {
		if iface.IP != nil {
			privateIPs = append(privateIPs, iface.IP.String())
		}
	}

	publicIPs := []string{}
	if !server.PublicNet.IPv4.IsUnspecified() {
		publicIPs = append(publicIPs, server.PublicNet.IPv4.IP.String())
	}
	if !server.PublicNet.IPv6.IsUnspecified() {
		publicIPs = append(publicIPs, server.PublicNet.IPv6.IP.String())
	}

	imageName := labelValue(server.Labels, opsImageNameLabelKey)

	return lepton.CloudInstance{
		ID:         strconv.FormatInt(server.ID, 10),
		Name:       server.Name,
		Status:     string(server.Status),
		Created:    server.Created.Format(time.RFC3339),
		PrivateIps: privateIPs,
		PublicIps:  publicIPs,
		Image:      imageName,
	}
}
