package lepton

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/go-errors/errors"
	"github.com/olekukonko/tablewriter"
)

var (
	localVolumeDir = path.Join(GetOpsHome(), "volumes")
)

const (
	// DefaultVolumeLabel is the default label of a volume created with mkfs
	DefaultVolumeLabel = "default"
	// MinimumVolumeSize is the minimum size of a volume created with mkfs (1 MB).
	MinimumVolumeSize = MByte
	volumeDelimiter   = ":"
)

var (
	errVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }
)

// NanosVolume information for nanos-managed volume
type NanosVolume struct {
	ID         string
	Name       string
	Label      string
	Data       string
	Size       string
	Path       string
	AttachedTo string
}

// CreateVolume creates volume for onprem image
// creates a volume named <name>:<uuid>
// where <uuid> is generated on creation
// also creates a symlink to volume label at <name>:<label>
// DefaultVolumeLabel will be used when label is not set
// TODO investigate symlinked volume interaction with image
// TODO symlink or hardlink
func (op *OnPrem) CreateVolume(config *Config, name, label, data, size, provider string) error {
	var mnfPath string
	mkfsPath := config.Mkfs
	var mkfsCommand = NewMkfsCommand(mkfsPath)
	mkfsCommand.SetLabel(label)

	tmp := fmt.Sprintf("%s.raw", name)
	mnf := fmt.Sprintf("%s.manifest", name)
	tmpPath := path.Join(localVolumeDir, tmp)
	mkfsCommand.SetFileSystemPath(tmpPath)

	if data != "" {
		config.Dirs = append(config.Dirs, data)
		mnfPath = path.Join(localVolumeDir, mnf)
		err := buildVolumeManifest(config, mnfPath)
		if err != nil {
			return err
		}

		src, err := os.Open(mnfPath)
		if err != nil {
			return err
		}
		defer src.Close()
		mkfsCommand.SetStdin(src)
	}

	mkfsCommand.SetupCommand()
	err := mkfsCommand.Execute()
	// fmt.Printf("mkfs %s\n", strings.Join(mkfsCommand.GetArgs(), " "))
	if err != nil {
		return errors.Wrap(fmt.Errorf("mkfs %s: %v", strings.Join(mkfsCommand.GetArgs(), " "), err), 1)
	}

	uuid := mkfsCommand.GetUUID()
	fmt.Printf("volume: %s created with UUID %s and label %s\n", name, uuid, label)

	raw := fmt.Sprintf("%s%s%s.raw", name, volumeDelimiter, uuid)
	rawPath := path.Join(localVolumeDir, raw)
	err = os.Rename(tmpPath, rawPath)
	if err != nil {
		fmt.Printf("volume: UUID: failed adding UUID info for volume %s\n", name)
		fmt.Printf("rename the file to %s%s%s should you want to attach it by UUID\n", name, volumeDelimiter, uuid)
		fmt.Printf("symlink the file to %s%s%s should you want to attach it by label\n", name, volumeDelimiter, label)
	} else {
		symlinkVolume(name, uuid, label)
	}

	cleanUpVolumeManifest(mnfPath)
	return nil
}

// GetAllVolumes prints list of all onprem nanos-managed volumes
// TODO refactor to private, expose ListVolumes instead
func (op *OnPrem) GetAllVolumes(config *Config) error {
	vols, err := op.getAllVolumes()
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "UUID", "LABEL", "SIZE", "PATH"})
	for _, vol := range vols {
		var row []string
		row = append(row, vol.Name)
		row = append(row, vol.ID)
		row = append(row, vol.Label)
		row = append(row, vol.Size)
		row = append(row, vol.Path)
		table.Append(row)
	}
	table.Render()
	return nil
}

// getAllVolumes gets list of all onprem nanos-managed volumes
func (op *OnPrem) getAllVolumes() ([]NanosVolume, error) {
	var vols []NanosVolume
	mvols := make(map[string]NanosVolume)

	fi, err := ioutil.ReadDir(localVolumeDir)
	if err != nil {
		return vols, err
	}

	for _, info := range fi {
		if info.IsDir() {
			continue
		}

		// checks if volume has been scanned from its symlink
		_, ok := mvols[info.Name()]
		if ok {
			continue
		}

		link, err := os.Readlink(path.Join(localVolumeDir, info.Name()))
		if err != nil {
			nu := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), volumeDelimiter)
			mvols[info.Name()] = NanosVolume{
				ID:   nu[1],
				Name: nu[0],
				Size: bytes2Human(info.Size()),
				Path: path.Join(localVolumeDir, info.Name()),
			}
		} else {
			label := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), volumeDelimiter)[1]
			src, _ := os.Stat(link)
			nu := strings.Split(strings.TrimSuffix(src.Name(), ".raw"), volumeDelimiter)
			mvols[src.Name()] = NanosVolume{
				ID:    nu[1],
				Name:  nu[0],
				Label: label,
				Size:  bytes2Human(info.Size()),
				Path:  path.Join(localVolumeDir, src.Name()),
			}
		}
	}

	for _, vol := range mvols {
		vols = append(vols, vol)
	}
	return vols, nil
}

// isVolumeExists checks if onprem nanos-managed volume
// with the given name exists
func (op *OnPrem) isVolumeExists(name string) (bool, error) {
	fi, err := ioutil.ReadDir(localVolumeDir)
	if err != nil {
		return false, err
	}

	for _, vol := range fi {
		if vol.Name() == name {
			return true, nil
		}
	}

	return false, errVolumeNotFound(name)
}

// UpdateVolume updates nanos-managed volume label for attach/detach purposes
func (op *OnPrem) UpdateVolume(config *Config, name, label string) error {
	_, err := op.isVolumeExists(name)
	if err != nil {
		return err
	}

	oldpath := path.Join(localVolumeDir, name)
	newpath := path.Join(localVolumeDir, label)
	return os.Symlink(oldpath, newpath)
}

// DeleteVolume deletes nanos-managed volume
// if label is specified, deletes only the symlink instead
// TODO delete symlink given actual filename?
func (op *OnPrem) DeleteVolume(config *Config, name, label string) error {
	path := path.Join(localVolumeDir, name+".raw")
	err := os.Remove(path)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches volume to instance on `ops instance create`
// or `ops run --mounts`
// on `ops image create --mount`, it simply creates a mount path
// with the given volume label
// label can refer to volume UUID or volume label
// TODO unstub
func (op *OnPrem) AttachVolume(config *Config, image, name, label, mount string) error {
	return nil
}

// DetachVolume detaches volume
// TODO unstub
func (op *OnPrem) DetachVolume(config *Config, image, label string) error {
	return nil
}

// parseSize parses the size of the NanosVolume to human readable format.
// If the size value is empty, it returns 1 MB (the default size of volumes).
func (op *OnPrem) parseSize(vol NanosVolume) string {
	if vol.Size == "" {
		// return the default size of a volume
		return bytes2Human(MiByte)
	}
	bytes, err := parseBytes(vol.Size)
	if err != nil {
		fmt.Printf("warning: invalid size value for volume %s with UUID %s: %s\n", vol.Name, vol.ID, err.Error())
	}
	size := bytes2Human(bytes)
	return size
}

// buildVolumeManifest ...
func buildVolumeManifest(conf *Config, out string) error {
	m := &Manifest{
		children:    make(map[string]interface{}),
		debugFlags:  make(map[string]rune),
		environment: make(map[string]string),
	}

	for _, d := range conf.Dirs {
		err := m.AddRelativeDirectory(d)
		if err != nil {
			return err
		}
	}

	m.AddEnvironmentVariable("USER", "root")
	m.AddEnvironmentVariable("PWD", "/")
	for k, v := range conf.Env {
		m.AddEnvironmentVariable(k, v)
	}

	return ioutil.WriteFile(out, []byte(m.String()), 0644)
}

// cleanUpVolumeManifest ...
func cleanUpVolumeManifest(file string) error {
	if file == "" {
		return nil
	}
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	err = os.Remove(file)
	if err != nil {
		fmt.Printf("failed cleaning up temporary manifest file: %s\n", file)
		return err
	}
	return nil
}

// symlinkVolume ...
func symlinkVolume(name, uuid, label string) error {
	msg := fmt.Sprintf("volume: label: failed adding label info for volume %s\n", name)
	msg = fmt.Sprintf("%vsymlink the file to %s%s%s should you want to attach it by label\n", msg, name, volumeDelimiter, label)

	src := path.Join(localVolumeDir, fmt.Sprintf("%s%s%s.raw", name, volumeDelimiter, uuid))
	dst := path.Join(localVolumeDir, fmt.Sprintf("%s%s%s.raw", name, volumeDelimiter, label))

	_, err := os.Lstat(dst)
	if err == nil {
		err := os.Remove(dst)
		if err != nil {
			fmt.Println(msg)
			return err
		}
	}
	if err != nil && !os.IsNotExist(err) {
		fmt.Println(msg)
		return err
	}

	err = os.Symlink(src, dst)
	if err != nil {
		fmt.Println(msg)
		return err
	}
	fmt.Printf("volume: label: volume %s is labelled %s\n", name, label)
	return nil
}
