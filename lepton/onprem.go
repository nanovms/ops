package lepton

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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
func (p *OnPrem) CreateImage(ctx *Context) error {
	return fmt.Errorf("Operation not supported")
}

// ListImages on premise
func (p *OnPrem) ListImages() error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images")
	if _, err := os.Stat(imgpath); os.IsNotExist(err) {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Path", "Size"})
	table.SetHeaderColor(
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
			row = append(row, fmt.Sprintf("%v", info.Size()))
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
func (p *OnPrem) DeleteImage(imagename string) error {
	opshome := GetOpsHome()
	imgpath := path.Join(opshome, "images", imagename)
	err := os.Remove(imgpath)
	if err != nil {
		return err
	}
	return nil
}

// CreateInstance on premise
func (p *OnPrem) CreateInstance(ctx *Context) error {
	return fmt.Errorf("Operation not supported")
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *Context) error {
	return fmt.Errorf("Operation not supported")
}

// DeleteInstance from on premise
func (p *OnPrem) DeleteInstance(ctx *Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// GetInstanceLogs for onprem instance logs
func (p *OnPrem) GetInstanceLogs(ctx *Context, instancename string) error {
	return fmt.Errorf("Operation not supported")
}

// Initialize on prem provider
func (p *OnPrem) Initialize() error {
	return nil
}
