package lepton

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/go-errors/errors"
	"github.com/olekukonko/tablewriter"
)

var (
	localVolumeDir  = path.Join(GetOpsHome(), "volumes")
	localVolumeData = path.Join(GetOpsHome(), "volumes", "volumes.json")
)

var (
	errVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }
	errVolumeMounted  = func(id, image string) error {
		return errors.Errorf("volume with UUID %s is already mounted on %s", id, image)
	}
	errVolumeNotMounted = func(id, image string) error {
		return errors.Errorf("volume with UUID %s is not mounted on %s", id, image)
	}
	errMountOccupied = func(mount, image string) error {
		return errors.Errorf("path %s on image %s is mounted", mount, image)
	}
	errInvalidMountConfiguration = func(mount string) error { return errors.Errorf("%s: invalid mount configuration", mount) }
)

// VolumeService interface for volume related operations
type VolumeService interface {
	CreateVolume(name, data, size string, config *Config) (NanosVolume, error)
	GetAllVolume(config *Config) error
	GetVolume(id string, config *Config) (NanosVolume, error)
	DeleteVolume(id string, config *Config) error
	AttachVolume(image, volume, mount string, config *Config) error
	DetachVolume(image, volume string, config *Config) error
}

// LocalVolume is service for managing nanos-managed local volumes
type LocalVolume struct {
	store volumeStore
}

// volumeStore interface for managing nanos volumes
type volumeStore interface {
	Insert(NanosVolume) error
	Get(string) (NanosVolume, error)
	GetAll() ([]NanosVolume, error)
	Update(NanosVolume) error
	Delete(string) (NanosVolume, error)
}

// NanosVolume information for nanos-managed volume
type NanosVolume struct {
	ID         string
	Name       string
	Data       string
	Size       string
	Path       string
	AttachedTo string
}

// NewLocalVolume instantiates new Volume
func NewLocalVolume() *LocalVolume {
	return &LocalVolume{
		store: &JSONStore{path: localVolumeData},
	}
}

// CreateVolume creates local volume
func (v *LocalVolume) CreateVolume(name, data, size string, config *Config) (NanosVolume, error) {
	var cmd *exec.Cmd
	var vol NanosVolume
	mkfs := config.Mkfs
	raw := name + ".raw"
	mnf := name + ".manifest"
	rawPath := path.Join(localVolumeDir, raw)
	var args []string
	if data != "" {
		config.Dirs = append(config.Dirs, data)
		mnfPath := path.Join(localVolumeDir, mnf)
		err := buildVolumeManifest(config, mnfPath)
		if err != nil {
			return vol, err
		}
		args = append(args, rawPath)
		src, err := os.Open(mnfPath)
		if err != nil {
			return vol, err
		}
		defer src.Close()
		cmd = exec.Command(mkfs, args...)
		cmd.Stdin = src
	} else {
		args = append(args, "-e", rawPath)
		if size != "" {
			args = append(args, "-s", size)
		}
		cmd = exec.Command(mkfs, args...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return vol, err
	}
	uuid := uuidFromMKFS(out)

	vol = NanosVolume{
		ID:   uuid,
		Name: name,
		Data: data,
		Size: size,
		Path: rawPath,
	}
	err = v.store.Insert(vol)
	if err != nil {
		return vol, err
	}

	log.Printf("volume %s created with UUID %s", name, uuid)
	return vol, nil
}

// GetAllVolume gets list of all nanos-managed local volumes
func (v *LocalVolume) GetAllVolume(config *Config) error {
	vols, err := v.store.GetAll()
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "UUID", "PATH"})
	for _, vol := range vols {
		var row []string
		row = append(row, vol.Name)
		row = append(row, vol.ID)
		row = append(row, vol.Path)
		table.Append(row)
	}
	table.Render()
	return nil
}

// GetVolume gets nanos-managed local volume by its UUID
func (v *LocalVolume) GetVolume(id string, config *Config) (NanosVolume, error) {
	return v.store.Get(id)
}

// DeleteVolume deletes nanos-managed local volume by its UUID
func (v *LocalVolume) DeleteVolume(id string, config *Config) error {
	// delete from storage
	vol, err := v.store.Delete(id)
	if err != nil {
		return err
	}

	// delete actual file
	err = os.Remove(vol.Path)
	if err != nil {
		return err
	}

	return nil
}

// AttachVolume attaches local volume to a stopped instance
func (v *LocalVolume) AttachVolume(image, volume, mount string, config *Config) error {
	um := strings.Split(mount, ":")
	if len(um) < 2 {
		return errInvalidMountConfiguration(mount)
	}
	uuid := um[0]
	path := strings.TrimPrefix(um[1], "/")
	vol, err := v.GetVolume(uuid, config)
	if err != nil {
		return err
	}
	conf := config
	// check if mounted
	if conf.Mounts == nil {
		conf.Mounts = make(map[string]string)
	}
	_, ok := conf.Mounts[uuid]
	if ok {
		return errVolumeMounted(uuid, "")
	}
	for _, mnt := range conf.Mounts {
		if mnt == mount {
			return errMountOccupied(path, "")
		}
	}
	// rebuild config to add mount
	conf.Mounts[uuid] = path
	conf.RunConfig.Mounts = append(conf.RunConfig.Mounts, vol.Path)
	return nil
}

// DetachVolume detaches local volume to a stopped instance
func (v *LocalVolume) DetachVolume(image, volume string, config *Config) error {
	return nil
}

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
