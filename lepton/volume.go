package lepton

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/go-errors/errors"
)

var volumeDir = path.Join(GetOpsHome(), "volumes")
var volumeData = path.Join(GetOpsHome(), "volumes", "volumes.json")

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
	var args []string
	if data != "" {
		var manifest string
		// build manifest and return out path
		args = append(args, raw)
		args = append(args, "<")
		args = append(args, manifest)
	} else {
		args = append(args, "-e")
		args = append(args, raw)
	}
	if size != "" {
		args = append(args, "-s")
		args = append(args, size)
	}

	cmd := exec.Command(mkfs, args...)
	log.Printf("cmd: %s", cmd.String())
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
		Path:     path.Join(volumeDir, raw),
		Provider: provider,
	}
	err = v.store.Insert(vol)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	log.Printf("volume %s created with UUID %s", name, uuid)
	return nil
}

type volumeStore interface {
	Insert(volume) error
	Get(string) (volume, error)
	GetAll() ([]volume, error)
	Delete(string) error
}

type JSONStore struct {
	path string
}

func (s *JSONStore) Insert(v volume) error {
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	err = enc.Encode(v)
	if err != nil {
		return err
	}
	return nil
}

func (s *JSONStore) Get(name string) (volume, error) {
	return volume{}, nil
}

func (s *JSONStore) GetAll() ([]volume, error) {
	return []volume{}, nil
}

func (s *JSONStore) Delete(name string) error {
	return nil
}
