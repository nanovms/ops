package lepton

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/types"
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

// MatchedByQueries returns true if this volume matched one or more given query.
func (v NanosVolume) MatchedByQueries(query map[string]string) bool {
	matched := false
	for key, value := range query {
		switch key {
		case "id":
			matched = matched || (value == v.ID)
		case "name":
			matched = matched || (value == v.Name)
		case "label":
			matched = matched || (value == v.Label)
		case "data":
			matched = matched || (value == v.Data)
		case "size":
			matched = matched || (value == v.Size)
		case "path":
			matched = matched || (value == v.Path)
		case "attached_to":
			matched = matched || (value == v.AttachedTo)
		case "created_at":
			matched = matched || (value == v.CreatedAt)
		case "status":
			matched = matched || (value == v.Status)
		}
	}
	return matched
}

const (
	// DefaultVolumeLabel is the default label of a volume created with mkfs
	DefaultVolumeLabel = "default"

	// VolumeDelimiter is the reserved character used as delimiter between
	// volume name and uuid/label
	VolumeDelimiter = ":"
)

var (
	// ErrVolumeNotFound is error returned when a volume with a given id does not exist
	ErrVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }
)

// CreateLocalVolume creates volume on ops directory
// creates a volume named <name>:<uuid>
// where <uuid> is generated on creation
// also creates a symlink to volume label at <name>
// TODO investigate symlinked volume interaction with image
func CreateLocalVolume(config *types.Config, name, data, provider string) (NanosVolume, error) {
	var vol NanosVolume
	var mkfsCommand *fs.MkfsCommand

	if data != "" {
		config.Dirs = append(config.Dirs, data)
		m, err := buildVolumeManifest(config)
		if err != nil {
			return vol, err
		}

		mkfsCommand = fs.NewMkfsCommand(m)
	} else {
		mkfsCommand = fs.NewMkfsCommand(nil)
	}

	mkfsCommand.SetLabel(name)
	tmp := fmt.Sprintf("%s.raw", name)
	tmpPath := path.Join(config.VolumesDir, tmp)
	mkfsCommand.SetFileSystemPath(tmpPath)

	if config.BaseVolumeSz != "" {
		mkfsCommand.SetFileSystemSize(config.BaseVolumeSz)
	}

	err := mkfsCommand.Execute()
	if err != nil {
		return vol, errors.Wrap(err, 1)
	}

	uuid := mkfsCommand.GetUUID()

	raw := fmt.Sprintf("%s%s%s.raw", name, VolumeDelimiter, uuid)
	rawPath := path.Join(config.VolumesDir, raw)
	err = os.Rename(tmpPath, rawPath)
	if err != nil {
		fmt.Printf("volume: UUID: failed adding UUID info for volume %s\n", name)
		fmt.Printf("rename the file to %s%s%s should you want to attach it by UUID\n", name, VolumeDelimiter, uuid)
		fmt.Printf("symlink the file to %s should you want to attach it by label\n", name)
	}

	vol = NanosVolume{
		ID:    uuid,
		Name:  name,
		Label: name,
		Data:  data,
		Path:  rawPath,
	}
	return vol, nil
}

// buildVolumeManifest builds manifests for non-empty volume
func buildVolumeManifest(conf *types.Config) (*fs.Manifest, error) {
	m := fs.NewManifest("")

	for _, d := range conf.Dirs {
		err := m.AddRelativeDirectory(d)
		if err != nil {
			return nil, err
		}
	}

	m.AddEnvironmentVariable("USER", "root")
	m.AddEnvironmentVariable("PWD", "/")
	for k, v := range conf.Env {
		m.AddEnvironmentVariable(k, v)
	}

	return m, nil
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

// GetSizeInGb converts a string representation of a volume size to an integer number of GB
func GetSizeInGb(size string) (int, error) {
	f := func(c rune) bool {
		return !unicode.IsNumber(c)
	}
	unitsIndex := strings.IndexFunc(size, f)
	var mul int64
	if unitsIndex < 0 {
		mul = 1
		unitsIndex = len(size)
	} else if unitsIndex == 0 {
		return 0, errors.New("invalid size " + size)
	} else {
		units := strings.ToLower(size[unitsIndex:])
		if units == "k" {
			mul = 1024
		} else if units == "m" {
			mul = 1024 * 1024
		} else if units == "g" {
			mul = 1024 * 1024 * 1024
		} else {
			return 0, errors.New("invalid units " + units)
		}
	}
	intSize, err := strconv.ParseInt(size[:unitsIndex], 10, 64)
	if err != nil {
		return 0, err
	}
	intSize *= mul
	sizeInGb := intSize / (1024 * 1024 * 1024)
	if sizeInGb*1024*1024*1024 < intSize {
		sizeInGb++
	}
	return int(sizeInGb), nil
}
