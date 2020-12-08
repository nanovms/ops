package lepton

import (
	"encoding/json"
	"errors"
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

	instancename := c.CloudConfig.ImageName

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
	return nil, errors.New("un-implemented")
}

// GetInstances return all instances on prem
// TODO
func (p *OnPrem) GetInstances(ctx *Context) ([]CloudInstance, error) {
	return nil, errors.New("un-implemented")
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *Context) error {

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

	opshome := GetOpsHome()
	instances := path.Join(opshome, "instances")

	files, err := ioutil.ReadDir(instances)
	if err != nil {
		fmt.Println(err)
	}

	for _, f := range files {
		var rows []string

		fullpath := path.Join(instances, f.Name())
		body, err := ioutil.ReadFile(fullpath)
		if err != nil {
			fmt.Println(err)
		}

		var i instance
		if err := json.Unmarshal(body, &i); err != nil {
			return err
		}

		rows = append(rows, f.Name())
		rows = append(rows, i.Image)
		rows = append(rows, "Running")
		rows = append(rows, time2Human(f.ModTime()))

		privateIps := []string{"127.0.0.1"}

		rows = append(rows, strings.Join(privateIps, ","))
		rows = append(rows, i.portList())
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
		fmt.Println(err)
	}

	// yolo
	err = sysKill(pid)
	if err != nil {
		fmt.Println(err)
	}

	opshome := GetOpsHome()
	ipath := path.Join(opshome, "instances", instancename)
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
