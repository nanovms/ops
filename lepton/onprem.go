package lepton

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/olekukonko/tablewriter"
)

// OnPrem provider for ops
type OnPrem struct{}

// BuildImage for onprem
func (p *OnPrem) BuildImage(ctx *Context) (string, error) {
	c := ctx.config
	err := BuildImage(*c)
	return "", err
}

// BuildImageWithPackage for onprem
func (p *OnPrem) BuildImageWithPackage(ctx *Context, pkgpath string) (string, error) {
	c := ctx.config
	err := BuildImageFromPackage(pkgpath, *c)
	if err != nil {
		return "", err
	}
	return "", nil
}

// CreateImage on prem
// assumes local for now
func (p *OnPrem) CreateImage(ctx *Context) error {
	return fmt.Errorf("Operation not supported")
}

// ResizeImage resizes the lcoal image imagename. You should never
// specify a negative size.
func (p *OnPrem) ResizeImage(ctx *Context, imagename string, hbytes string) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", imagename)

	bytes, err := parseBytes(hbytes)
	if err != nil {
		return err
	}

	return os.Truncate(imgpath, bytes)
}

// ListImages on premise
func (p *OnPrem) ListImages(ctx *Context) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images")

	if _, err := os.Stat(imgpath); os.IsNotExist(err) {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Path", "Size", "CreatedAt"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetRowLine(true)
	err := filepath.Walk(imgpath, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()

		if len(name) > 4 && strings.LastIndex(info.Name(), ".img") == len(name)-4 {
			var row []string
			row = append(row, info.Name())
			row = append(row, hostpath)
			row = append(row, bytes2Human(info.Size()))
			row = append(row, time2Human(info.ModTime()))
			table.Append(row)
		}
		return nil
	})
	if err != nil {
		return err
	}
	table.Render()
	return nil
}

// DeleteImage on premise
func (p *OnPrem) DeleteImage(ctx *Context, imagename string) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", imagename)
	err := os.Remove(imgpath)
	if err != nil {
		return err
	}
	return nil
}

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
	err = syscall.Kill(pid, 9)
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

// GetInstanceLogs for onprem instance logs
func (p *OnPrem) GetInstanceLogs(ctx *Context, instancename string, watch bool) error {

	body, err := ioutil.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(string(body))

	return nil
}

// Initialize on prem provider
func (p *OnPrem) Initialize() error {
	return nil
}
