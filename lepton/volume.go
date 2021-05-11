package lepton

import (
	"fmt"
	"os"
	"path"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/fs"
	"github.com/nanovms/ops/log"
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

const (
	// DefaultVolumeLabel is the default label of a volume created with mkfs
	DefaultVolumeLabel = "default"

	// VolumeDelimiter is the reserved character used as delimiter between
	// volume name and uuid/label
	VolumeDelimiter = ":"
)

var (
	errVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }
)

// CreateLocalVolume creates volume on ops directory
// creates a volume named <name>:<uuid>
// where <uuid> is generated on creation
// also creates a symlink to volume label at <name>
// TODO investigate symlinked volume interaction with image
func CreateLocalVolume(config *types.Config, name, data, size, provider string) (NanosVolume, error) {
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

	if config.BaseVolumeSz != "" && size == "" {
		mkfsCommand.SetFileSystemSize(config.BaseVolumeSz)
	} else if size != "" {
		mkfsCommand.SetFileSystemSize(size)
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
	} else {
		symlinkVolume(config.VolumesDir, name, uuid)
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

// symlinkVolume creates a symlink to volume that acts as volume label
// if label of the same name exists for a volume, removes the label from the older volume
// and assigns it to the newly created volume
func symlinkVolume(dir, name, uuid string) error {
	msg := fmt.Sprintf("volume: label: failed adding label info for volume %s\n", name)
	msg = fmt.Sprintf("%vsymlink the file to %s should you want to attach it by label\n", msg, name)

	src := path.Join(dir, fmt.Sprintf("%s%s%s.raw", name, VolumeDelimiter, uuid))
	dst := path.Join(dir, fmt.Sprintf("%s.raw", name))

	_, err := os.Lstat(dst)
	if err == nil {
		err := os.Remove(dst)
		if err != nil {
			log.Error(msg)
			log.Error(err.Error())
			return err
		}
	}
	if err != nil && !os.IsNotExist(err) {
		log.Error(msg)
		log.Error(err.Error())
		return err
	}

	err = os.Symlink(src, dst)
	if err != nil {
		log.Error(msg)
		log.Error(err.Error())
		return err
	}
	return nil
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
