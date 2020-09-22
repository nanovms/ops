package lepton

import (
	"encoding/json"
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

// Volume is service for managing nanos-managed volumes
type Volume struct {
	config *Config
	store  volumeStore
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
	Provider   string // TODO change to enum/custom type
}

// NewVolume instantiates new Volume
func NewVolume(config *Config) *Volume {
	return &Volume{
		config: config,
		store:  &JSONStore{path: localVolumeData},
	}
}

// Create creates volume
func (v *Volume) Create(name, data, size, provider string) error {
	var cmd *exec.Cmd
	mkfs := v.config.Mkfs
	raw := name + ".raw"
	mnf := name + ".manifest"
	rawPath := path.Join(localVolumeDir, raw)
	var args []string
	if data != "" {
		v.config.Dirs = append(v.config.Dirs, data)
		mnfPath := path.Join(localVolumeDir, mnf)
		err := buildVolumeManifest(v.config, mnfPath)
		if err != nil {
			return err
		}
		args = append(args, rawPath)
		src, err := os.Open(mnfPath)
		if err != nil {
			return err
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
		return errors.Wrap(err, 1)
	}
	uuid := uuidFromMKFS(out)

	vol := NanosVolume{
		ID:       uuid,
		Name:     name,
		Data:     data,
		Size:     size,
		Path:     rawPath,
		Provider: provider,
	}
	err = v.store.Insert(vol)
	if err != nil {
		log.Println("insert", err)
		return errors.Wrap(err, 1)
	}

	log.Printf("volume %s created with UUID %s", name, uuid)
	return nil
}

// GetAll gets list of all nanos-managed volumes
func (v *Volume) GetAll() error {
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

// Get gets nanos-managed volume by its UUID
func (v *Volume) Get(id string) (NanosVolume, error) {
	return v.store.Get(id)
}

// Update updates nanos-managed volume for attach/detach purposes
func (v *Volume) Update(id string, vol NanosVolume) error {
	cur, err := v.Get(id)
	if err != nil {
		return err
	}
	// TODO more general update
	cur.AttachedTo = vol.AttachedTo
	err = v.store.Update(cur)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes nanos-managed volume by its UUID
func (v *Volume) Delete(id string) error {
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

// Attach attaches volume to a stopped instance
func (v *Volume) Attach(image, id, mount string) error {
	mount = strings.TrimPrefix(mount, "/")
	mnf := image + ".json"
	mnfPath := path.Join(localManifestDir, mnf)

	b, err := ioutil.ReadFile(mnfPath)
	if err != nil {
		return err
	}

	// unmarshal manifest to config
	var conf Config
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return err
	}
	conf.NightlyBuild = v.config.NightlyBuild

	// preserve default
	conf.BuildDir = v.config.BuildDir

	// check if mounted
	if conf.Mounts == nil {
		conf.Mounts = make(map[string]string)
	}
	_, ok := conf.Mounts[id]
	if ok {
		return errVolumeMounted(id, image)
	}
	for _, mnt := range conf.Mounts {
		if mnt == mount {
			return errMountOccupied(mount, image)
		}
	}
	// rebuild config to add mount
	conf.Mounts[id] = mount
	// rebuild image and manifest to add mount
	err = rebuildImage(conf)
	if err != nil {
		return err
	}
	return nil
}

// AttachOnRun attaches volume to instance on `ops run`
func (v *Volume) AttachOnRun(mount string) error {
	um := strings.Split(mount, ":")
	if len(um) < 2 {
		return errInvalidMountConfiguration(mount)
	}
	uuid := um[0]
	path := strings.TrimPrefix(um[1], "/")
	vol, err := v.Get(uuid)
	if err != nil {
		return err
	}
	conf := v.config
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
