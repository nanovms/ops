package lepton

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/go-errors/errors"
	"github.com/olekukonko/tablewriter"
)

// NanosVolume information for nanos-managed volume
type NanosVolume struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Label      string `json:"label"`
	Data       string `json:"data"`
	Size       string `json:"size"`
	Path       string `json:"path"`
	AttachedTo string `json:"attached_to"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"`
}

// CreateLocalVolume creates volume on ops directoy
// creates a volume named <name>:<uuid>
// where <uuid> is generated on creation
// also creates a symlink to volume label at <name>
// TODO investigate symlinked volume interaction with image
func CreateLocalVolume(config *Config, name, data, size, provider string) (NanosVolume, error) {
	var vol NanosVolume
	var mnfPath string
	mkfsPath := config.Mkfs
	var mkfsCommand = NewMkfsCommand(mkfsPath)
	mkfsCommand.SetLabel(name)

	tmp := fmt.Sprintf("%s.raw", name)
	mnf := fmt.Sprintf("%s.manifest", name)
	tmpPath := path.Join(config.BuildDir, tmp)
	mkfsCommand.SetFileSystemPath(tmpPath)

	if data != "" {
		config.Dirs = append(config.Dirs, data)
		mnfPath = path.Join(config.BuildDir, mnf)
		err := buildVolumeManifest(config, mnfPath)
		if err != nil {
			return vol, err
		}

		src, err := os.Open(mnfPath)
		if err != nil {
			return vol, err
		}
		defer src.Close()
		mkfsCommand.SetStdin(src)
	} else {
		mkfsCommand.SetEmptyFileSystem()
	}

	if config.BaseVolumeSz != "" {
		mkfsCommand.SetFileSystemSize(config.BaseVolumeSz)
	}

	mkfsCommand.SetupCommand()
	err := mkfsCommand.Execute()
	if err != nil {
		return vol, errors.Wrap(fmt.Errorf("mkfs %s: %v", strings.Join(mkfsCommand.GetArgs(), " "), err), 1)
	}

	uuid := mkfsCommand.GetUUID()

	raw := fmt.Sprintf("%s%s%s.raw", name, VolumeDelimiter, uuid)
	rawPath := path.Join(config.BuildDir, raw)
	err = os.Rename(tmpPath, rawPath)
	if err != nil {
		fmt.Printf("volume: UUID: failed adding UUID info for volume %s\n", name)
		fmt.Printf("rename the file to %s%s%s should you want to attach it by UUID\n", name, VolumeDelimiter, uuid)
		fmt.Printf("symlink the file to %s should you want to attach it by label\n", name)
	} else {
		symlinkVolume(config.BuildDir, name, uuid)
	}

	cleanUpVolumeManifest(mnfPath)
	vol = NanosVolume{
		ID:    uuid,
		Name:  name,
		Label: name,
		Data:  data,
		Path:  rawPath,
	}
	return vol, nil
}

// PrintVolumesList writes into console a table with volumes details
func PrintVolumesList(volumes *[]NanosVolume) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"UUID", "Name", "Status", "Size (GB)", "Location", "Created", "Attached"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
	)
	table.SetRowLine(true)

	for _, vol := range *volumes {
		var row []string
		row = append(row, vol.ID)
		row = append(row, vol.Name)
		row = append(row, vol.Status)
		row = append(row, vol.Size)
		row = append(row, vol.Path)
		row = append(row, vol.CreatedAt)
		row = append(row, vol.AttachedTo)
		table.Append(row)
	}

	table.Render()
}
