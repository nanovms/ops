//go:build ibm || !onlyprovider

package ibm

import (
	"bytes"
	"encoding/json"
	"errors"
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

func extractRegionFromZone(zone string) string {
	s := strings.Split(zone, "-")
	return s[0] + "-" + s[1]
}

// CreateInstance - Creates instance on IBM Cloud Platform
func (v *IBM) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()
	zone := c.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	imgs, err := v.GetImages(ctx)
	if err != nil {
		fmt.Println(err)
	}

	config := ctx.Config()

	imgID := ""
	for i := 0; i < len(imgs); i++ {
		if imgs[i].Name == config.CloudConfig.ImageName {
			imgID = imgs[i].ID
		}
	}
	if imgID == "" {
		return errors.New("can't find image")
	}

	uri := "https://" + region + ".iaas.cloud.ibm.com/v1/instances?version=2023-02-28&generation=2"

	vpcID := v.getDefaultVPC(region)
	subnetID := v.getDefaultSubnet(region)

	t := time.Now().Unix()
	st := strconv.FormatInt(t, 10)
	instName := config.CloudConfig.ImageName + "-" + st

	stuff := `{
  "boot_volume_attachment": {
    "volume": {
      "name": "my-boot-volume",
      "profile": {
        "name": "general-purpose"
      }
    }
  },
  "image": {
    "id": "` + imgID + `"
  },
  "name": "` + instName + `",
  "primary_network_interface": {
    "name": "my-network-interface",
    "subnet": {
      "id": "` + subnetID + `"
    }
  },
  "profile": {
    "name": "bx2-2x8"
  },
  "vpc": {
    "id": "` + vpcID + `"
  },
  "zone": {
    "name": "` + zone + `"
  }
}`

	reqBody := []byte(stuff)

	client := &http.Client{}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)
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

	return nil
}

// GetInstanceByName returns instance with given name
func (v *IBM) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// InstancesListResponse is the set of instances available from IBM in an
// images list call.
type InstancesListResponse struct {
	Instances []Instance `json:"instance"`
}

// Instance represents a virtual server instance.
type Instance struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetInstances return all instances on IBM
func (v *IBM) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	c := ctx.Config()
	zone := c.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	uri := "https://" + region + ".iaas.cloud.ibm.com/v1/instances?version=2023-02-28&generation=2"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.iam)

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

	ilr := &InstancesListResponse{}
	err = json.Unmarshal(body, &ilr)
	if err != nil {
		fmt.Println(err)
	}

	var cloudInstances []lepton.CloudInstance

	for _, instance := range ilr.Instances {
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
func (v *IBM) ListInstances(ctx *lepton.Context) error {
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

// DeleteInstance deletes instance from IBM
func (v *IBM) DeleteInstance(ctx *lepton.Context, instanceID string) error {
	c := ctx.Config()
	zone := c.CloudConfig.Zone

	region := extractRegionFromZone(zone)

	uri := "https://" + region + ".iaas.cloud.ibm.com/v1/instances/$instance_id?version=2023-02-28&generation=2"

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", v.iam)
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
func (v *IBM) RebootInstance(ctx *lepton.Context, instanceName string) error {
	return fmt.Errorf("operation not supported")
}

// StartInstance starts an instance in IBM
func (v *IBM) StartInstance(ctx *lepton.Context, instanceID string) error {
	return nil
}

// StopInstance halts instance from v
func (v *IBM) StopInstance(ctx *lepton.Context, instanceID string) error {
	// POST https://api.IBM.com/v4/IBM/instances/{IBMId}/shutdown
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (v *IBM) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// GetInstanceLogs gets instance related logs
// https://cloud.ibm.com/docs/vpc?topic=vpc-vsi_is_connecting_console&interface=api
func (v *IBM) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {
	return "", nil
}

// InstanceStats show metrics for instances on ibm.
func (p *IBM) InstanceStats(ctx *lepton.Context) error {
	return errors.New("currently not avilable")
}
