package lepton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"

	"github.com/go-errors/errors"
)

var volumeDir = path.Join(GetOpsHome(), "volumes")
var volumeData = path.Join(GetOpsHome(), "volumes", "volumes.json")

var errVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }

type Volume struct {
	config *Config
	store  volumeStore
}

type volume struct {
	ID       string
	Name     string
	Data     string
	Size     string
	Path     string
	Provider string // TODO change to enum/custom type
}

func NewVolume(config *Config) *Volume {
	return &Volume{
		config: config,
		store:  &JSONStore{path: volumeData},
	}
}

// Create creates volume
func (v *Volume) Create(name, data, size, provider, mkfs string) error {
	raw := name + ".raw"
	rawPath := path.Join(volumeDir, raw)
	var args []string
	if data != "" {
		var manifest string
		// build manifest and return out path
		args = append(args, rawPath)
		args = append(args, "<")
		args = append(args, manifest)
	} else {
		args = append(args, "-e")
		args = append(args, rawPath)
	}
	if size != "" {
		args = append(args, "-s")
		args = append(args, size)
	}

	cmd := exec.Command(mkfs, args...)
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	uuid := uuidFromMKFS(out)

	vol := volume{
		ID:       uuid,
		Name:     name,
		Data:     data,
		Size:     size,
		Path:     rawPath,
		Provider: provider,
	}
	err = v.store.Insert(vol)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	log.Printf("volume %s created with UUID %s", name, uuid)
	return nil
}

func (v *Volume) GetAll() error {
	vols, err := v.store.GetAll()
	if err != nil {
		return err
	}
	fmt.Println("NAME\t\t\tUUID\t\t\tPATH")
	for _, vol := range vols {
		fmt.Printf("%s\t\t\t%s\t\t\t%s\n", vol.Name, vol.ID, vol.Path)
	}
	return nil
}

func (v *Volume) Get(id string) (volume, error) {
	return v.store.Get(id)
}

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

type volumeStore interface {
	Insert(volume) error
	Get(string) (volume, error)
	GetAll() ([]volume, error)
	Delete(string) (volume, error)
}

type JSONStore struct {
	path string
}

func (s *JSONStore) Insert(vol volume) error {
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	err = enc.Encode(vol)
	if err != nil {
		return err
	}
	return nil
}

func (s *JSONStore) Get(id string) (volume, error) {
	var vol volume
	f, err := os.Open(s.path)
	if err != nil {
		return vol, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		err = dec.Decode(&vol)
		if err == io.EOF {
			break
		}
		if err != nil {
			return vol, err
		}
		if vol.ID == id {
			return vol, nil
		}
	}
	return vol, errVolumeNotFound(id)
}

func (s *JSONStore) GetAll() ([]volume, error) {
	var volumes []volume
	f, err := os.Open(s.path)
	if err != nil {
		return volumes, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var vol volume
		err = dec.Decode(&vol)
		if err == io.EOF {
			break
		}
		if err != nil {
			return volumes, err
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

func (s *JSONStore) Delete(id string) (volume, error) {
	var volumes []volume
	var vol volume
	f, err := os.Open(s.path)
	if err != nil {
		return vol, err
	}
	dec := json.NewDecoder(f)
	for {
		var cur volume
		err = dec.Decode(&cur)
		if err == io.EOF {
			break
		}
		if err != nil {
			return vol, err
		}
		if cur.ID == id {
			vol = cur
			continue
		}
		volumes = append(volumes, cur)
	}
	if reflect.DeepEqual(vol, volume{}) {
		return vol, errVolumeNotFound(id)
	}
	f.Close()

	f, _ = os.OpenFile(s.path, os.O_RDWR|os.O_TRUNC, 0644)
	buf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(buf)
	for _, vol := range volumes {
		err = enc.Encode(vol)
		if err != nil {
			return vol, err
		}
	}
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return vol, err
	}
	return vol, nil
}
