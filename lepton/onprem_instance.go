package lepton

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// CreateInstance on premise
// assumes local
func (p *OnPrem) CreateInstance(ctx *Context) error {
	c := ctx.config

	hypervisor := HypervisorInstance()
	if hypervisor == nil {
		fmt.Println("No hypervisor found on $PATH")
		fmt.Println("Please install OPS using curl https://ops.city/get.sh -sSfL | sh")
		os.Exit(1)
	}

	instancename := c.RunConfig.InstanceName

	if instancename == "" {
		instancename = c.CloudConfig.ImageName
	}

	fmt.Printf("booting %s ...\n", instancename)

	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", instancename)

	c.RunConfig.BaseName = instancename
	c.RunConfig.Imagename = imgpath
	c.RunConfig.OnPrem = true

	hypervisor.Start(&c.RunConfig)

	return nil
}

// GetInstanceByID returns the instance with the id passed by argument if it exists
func (p *OnPrem) GetInstanceByID(ctx *Context, id string) (*CloudInstance, error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		if i.Image == id {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("instance with id \"%s\" not found", id)
}

// GetInstances return all instances on prem
func (p *OnPrem) GetInstances(ctx *Context) (instances []CloudInstance, err error) {
	opshome := GetOpsHome()
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

		instances = append(instances, CloudInstance{
			Name:       f.Name(),
			Image:      i.Image,
			Status:     "Running",
			Created:    Time2Human(f.ModTime()),
			PrivateIps: []string{"127.0.0.1"},
			PublicIps:  strings.Split(i.portList(), ","),
		})
	}

	return
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PID", "Name", "Status", "Created", "Private Ips", "Port"})
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
func (p *OnPrem) StartInstance(ctx *Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// StopInstance from on premise
func (p *OnPrem) StopInstance(ctx *Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// DeleteInstance from on premise
func (p *OnPrem) DeleteInstance(ctx *Context, instancename string) error {

	pid, err := strconv.Atoi(instancename)
	if err != nil {
		instance, err := p.GetInstanceByID(ctx, instancename)
		if err == nil {
			pid, _ = strconv.Atoi(instance.Name)
		}
	}

	if pid == 0 {
		fmt.Printf("did not find pid of instance \"%s\"\n", instancename)
		return nil
	}

	err = sysKill(pid)
	if err != nil {
		fmt.Println(err)
	}

	opshome := GetOpsHome()
	ipath := path.Join(opshome, "instances", strconv.Itoa(pid))
	err = os.Remove(ipath)
	if err != nil {
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *OnPrem) PrintInstanceLogs(ctx *Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs for onprem instance logs
func (p *OnPrem) GetInstanceLogs(ctx *Context, instancename string) (string, error) {

	body, err := ioutil.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(body), nil
}
