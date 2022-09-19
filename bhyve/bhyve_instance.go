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
	"strconv"
)

// returns immediately for background job
func execCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("/usr/local/bin/bash", "-c", cmdStr)
	err = cmd.Run()
	if err != nil {
		return
	}

	output = ""
	return output, nil
}

// blocks
func blockExecCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("/usr/local/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()

	output = string(out)
	return output, err
}

// CreateInstance - Creates instance on Bhyve - only local support for now
// need to load nmdm module: kldload nmdm
func (p *Bhyve) CreateInstance(ctx *lepton.Context) error {

	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	i := len(instances)
	si := strconv.Itoa(i)

	tapN := "tap" + si
	nm := "/dev/nmdm" + si + "A"

	cmd := "ifconfig " + tapN + " create"
	out, err := blockExecCmd(cmd)
	if err != nil {
		fmt.Println(out)
		fmt.Println(err)
		return err
	}
	fmt.Println(out)

	cmd = "sysctl net.link.tap.up_on_open=1"
	out, err = blockExecCmd(cmd)
	if err != nil {
		fmt.Println(out)
		fmt.Println(err)
		return err
	}
	fmt.Println(out)

	if i == 0 {
		cmd = "ifconfig bridge0 create"
		out, err = blockExecCmd(cmd)
		if err != nil {
			fmt.Println(out)
			fmt.Println(err)
			return err
		}
		fmt.Println(out)

		cmd = "ifconfig bridge0 addm em0 addm " + tapN
		out, err = blockExecCmd(cmd)
		if err != nil {
			fmt.Println(out)
			fmt.Println(err)
			return err
		}
		fmt.Println(out)

	} else {

		cmd = "ifconfig bridge0 addm " + tapN
		out, err = blockExecCmd(cmd)
		if err != nil {
			fmt.Println(out)
			fmt.Println(err)
			return err
		}
		fmt.Println(out)

	}

	cmd = "ifconfig bridge0 up"
	out, err = blockExecCmd(cmd)
	if err != nil {
		fmt.Println(out)
		fmt.Println(err)
		return err
	}
	fmt.Println(out)

	c := ctx.Config()

	rimg := path.Join(lepton.GetOpsHome(), "images", c.CloudConfig.ImageName)

	cmd = "bhyve -AHP -s 0:0,hostbridge -s 1:0,virtio-blk," + rimg + " " +
	 "-s 2:0,virtio-net," + tapN + " -s 3:0,virtio-rnd -s 31:0,lpc " + 
	 "-l bootrom,/usr/local/share/uefi-firmware/BHYVE_UEFI.fd " +
	 "-l com1," + nm + " " + c.CloudConfig.ImageName + " >> /tmp/" + c.CloudConfig.ImageName + ".log &"

	 fmt.Println(cmd)

	out, err = execCmd(cmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
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
func (p *Bhyve) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	cmd := "ls /dev/vmm"
	out, err := blockExecCmd(cmd)
	if err != nil {
		return nil, err
	}

	ins := strings.Split(out, "\n")

	// scrape ip from logs or do arp lookup

	for i:=0; i<len(ins); i++ {
		iname := strings.TrimSpace(ins[i])

		if iname != "" {
			instances = append(instances, lepton.CloudInstance{
				Name: iname,
			})
		}
	}

	return instances, nil
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
// would use simple serial redirection but there is a bug with backgrounding it:
// https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=264038
// instead we use the null modem kernel module
// cu -l /dev/nmdm0B -s 9600
func (p *Bhyve) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {

	body, err := ioutil.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		log.Fatal(err)
	}

	return string(body), nil
}
