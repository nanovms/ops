package onprem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/olekukonko/tablewriter"
)

// CreateInstance on premise
// assumes local
func (p *OnPrem) CreateInstance(ctx *lepton.Context) error {
	c := ctx.Config()

	imageName := c.CloudConfig.ImageName

	if matched, _ := regexp.Match(`.img$`, []byte(imageName)); !matched {
		c.CloudConfig.ImageName = imageName + ".img"
	}

	if _, err := os.Stat(path.Join(lepton.GetOpsHome(), "images", c.CloudConfig.ImageName)); os.IsNotExist(err) {
		return fmt.Errorf("image \"%s\" not found", imageName)
	}

	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		fmt.Println("No hypervisor found on $PATH")
		fmt.Println("Please install OPS using curl https://ops.city/get.sh -sSfL | sh")
		os.Exit(1)
	}

	if c.RunConfig.InstanceName == "" {
		c.RunConfig.InstanceName = strings.Split(c.CloudConfig.ImageName, ".")[0]
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.InstanceName)

	opshome := lepton.GetOpsHome()
	imgpath := path.Join(opshome, "images", c.CloudConfig.ImageName)

	c.RunConfig.Imagename = imgpath
	c.RunConfig.Background = true

	err := hypervisor.Start(&c.RunConfig)
	if err != nil {
		return err
	}

	pid, err := hypervisor.PID()
	if err != nil {
		return err
	}

	instances := path.Join(opshome, "instances")

	i := instance{
		Instance: c.RunConfig.InstanceName,
		Image:    c.RunConfig.Imagename,
		Ports:    c.RunConfig.Ports,
	}

	d1, err := json.Marshal(i)
	if err != nil {
		log.Error(err)
	}

	err = ioutil.WriteFile(instances+"/"+pid, d1, 0644)
	if err != nil {
		log.Error(err)
	}

	return nil
}

// GetInstanceByName returns instance with given name
func (p *OnPrem) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		if i.Name == name {
			return &i, nil
		}
	}

	return nil, lepton.ErrInstanceNotFound(name)
}

// GetInstances return all instances on prem
func (p *OnPrem) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	opshome := lepton.GetOpsHome()
	instancesPath := path.Join(opshome, "instances")

	files, err := ioutil.ReadDir(instancesPath)
	if err != nil {
		return
	}

	for _, f := range files {
		fullpath := path.Join(instancesPath, f.Name())
		body, err := ioutil.ReadFile(fullpath)
		if err != nil {
			return nil, err
		}

		var i instance
		if err := json.Unmarshal(body, &i); err != nil {
			return nil, err
		}

		instances = append(instances, lepton.CloudInstance{
			ID:         f.Name(),
			Name:       i.Instance,
			Image:      i.Image,
			Status:     "Running",
			Created:    lepton.Time2Human(f.ModTime()),
			PrivateIps: []string{"127.0.0.1"},
			PublicIps:  strings.Split(i.portList(), ","),
		})
	}

	return
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PID", "Name", "Image", "Status", "Created", "Private Ips", "Port"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
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
		rows = append(rows, i.Image)
		rows = append(rows, i.Status)
		rows = append(rows, i.Created)
		rows = append(rows, strings.Join(i.PrivateIps, ","))
		rows = append(rows, strings.Join(i.PublicIps, ","))

		table.Append(rows)
	}

	table.Render()

	return nil

}

// StartInstance from on premise
func (p *OnPrem) StartInstance(ctx *lepton.Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// StopInstance from on premise
func (p *OnPrem) StopInstance(ctx *lepton.Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// DeleteInstance from on premise
func (p *OnPrem) DeleteInstance(ctx *lepton.Context, instancename string) error {

	var pid int
	instance, err := p.GetInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	pid, _ = strconv.Atoi(instance.ID)

	if pid == 0 {
		fmt.Printf("did not find pid of instance \"%s\"\n", instancename)
		return nil
	}

	err = sysKill(pid)
	if err != nil {
		log.Error(err)
	}

	opshome := lepton.GetOpsHome()
	ipath := path.Join(opshome, "instances", strconv.Itoa(pid))
	err = os.Remove(ipath)
	if err != nil {
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *OnPrem) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs for onprem instance logs
func (p *OnPrem) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {

	body, err := ioutil.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		log.Fatal(err)
	}

	return string(body), nil
}
