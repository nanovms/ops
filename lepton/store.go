package lepton

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"reflect"
)

// JSONStore implements volumeStore
// TODO probably use a more established KV-store
type JSONStore struct {
	path string
}

// Insert inserts volume data
func (s *JSONStore) Insert(vol NanosVolume) error {
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

// Get gets volume data of a given UUID
func (s *JSONStore) Get(id string) (NanosVolume, error) {
	var vol NanosVolume
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

// GetAll gets all volume data
func (s *JSONStore) GetAll() ([]NanosVolume, error) {
	var volumes []NanosVolume
	f, err := os.Open(s.path)
	if err != nil {
		return volumes, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var vol NanosVolume
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

// Update updates a given volume
func (s *JSONStore) Update(v NanosVolume) error {
	var volumes []NanosVolume
	var vol NanosVolume
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(f)
	for {
		var cur NanosVolume
		err = dec.Decode(&cur)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if cur.ID == v.ID {
			cur = v
			vol = v
		}
		volumes = append(volumes, cur)
	}
	if reflect.DeepEqual(vol, NanosVolume{}) {
		return errVolumeNotFound(v.ID)
	}
	f.Close()

	f, _ = os.OpenFile(s.path, os.O_RDWR|os.O_TRUNC, 0644)
	buf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(buf)
	for _, vol := range volumes {
		err = enc.Encode(vol)
		if err != nil {
			return err
		}
	}
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes volume of a given UUID
func (s *JSONStore) Delete(id string) (NanosVolume, error) {
	var volumes []NanosVolume
	var vol NanosVolume
	f, err := os.Open(s.path)
	if err != nil {
		return vol, err
	}
	dec := json.NewDecoder(f)
	for {
		var cur NanosVolume
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
	if reflect.DeepEqual(vol, NanosVolume{}) {
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
