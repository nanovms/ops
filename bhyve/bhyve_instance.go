package bhyve

import (
	"github.com/nanovms/ops/lepton"

	"github.com/olekukonko/tablewriter"

	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"os"
	"path"
	"strings"
)

func execCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("/usr/local/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	
	output = string(out)
	return
}

// CreateInstance - Creates instance on Bhyve - only local support for now
func (p *Bhyve) CreateInstance(ctx *lepton.Context) error {

	c := ctx.Config()

	rimg := path.Join(lepton.GetOpsHome(), "images", c.CloudConfig.ImageName)

	cmd := "bhyve -AHP -s 0:0,hostbridge -s 1:0,virtio-blk," + rimg + " " +
	 "-s 2:0,virtio-net,tap0 -s 3:0,virtio-rnd -s 31:0,lpc " + 
	 "-l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd " +
	 "-l com1,/dev/nmdm0A nanos >> /tmp/" + c.CloudConfig.ImageName + ".log 2>&1 &"

	out, err := execCmd(cmd)
	fmt.Println(out)

	return err
}

// ListInstances lists instances on local Bhyve
func (p *Bhyve) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Status", "Created", "Private Ips", "Public Ips", "Image"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	for _, instance := range instances {
		var rows []string
		rows = append(rows, instance.Name)
		rows = append(rows, instance.Status)
		rows = append(rows, instance.Created)
		rows = append(rows, strings.Join(instance.PrivateIps, ","))
		rows = append(rows, strings.Join(instance.PublicIps, ","))
		rows = append(rows, instance.Image)
		table.Append(rows)
	}
	table.Render()
	return nil
}

// GetInstanceByName returns instance with given name
func (p *Bhyve) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	return nil, nil
}

// GetInstances return all instances on Bhyve
func (p *Bhyve) GetInstances(ctx *lepton.Context) ([]lepton.CloudInstance, error) {
	return nil, nil
}

// DeleteInstance deletes instance from Bhyve
func (p *Bhyve) DeleteInstance(ctx *lepton.Context, instancename string) error {

	cmd := "bhyvectl --destroy --vm=" + instancename

	out, err := execCmd(cmd)
	fmt.Println(out)

	return err
}

// StartInstance starts an instance in Bhyve
func (p *Bhyve) StartInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// StopInstance stops instance
func (p *Bhyve) StopInstance(ctx *lepton.Context, instancename string) error {
	// bhyvectl --destroy --vm=nanos
	return nil
}

// ResetInstance resets instance
func (p *Bhyve) ResetInstance(ctx *lepton.Context, instancename string) error {
	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *Bhyve) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	return nil
}

// GetInstanceLogs gets instance related logs
func (p *Bhyve) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {

	body, err := ioutil.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		log.Fatal(err)
	}

	return string(body), nil
}
