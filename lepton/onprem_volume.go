package lepton

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/go-errors/errors"
)

var (
	// LocalVolumeDir is the default local volume directory
	LocalVolumeDir = path.Join(GetOpsHome(), "volumes")
)

const (
	// DefaultVolumeLabel is the default label of a volume created with mkfs
	DefaultVolumeLabel = "default"
	// MinimumVolumeSize is the minimum size of a volume created with mkfs (1 MB).
	MinimumVolumeSize = MByte
	// VolumeDelimiter is the reserved character used as delimiter between
	// volume name and uuid/label
	VolumeDelimiter = ":"
)

var (
	errVolumeNotFound = func(id string) error { return errors.Errorf("volume with UUID %s not found", id) }
)

// CreateVolume creates volume for onprem image
func (op *OnPrem) CreateVolume(ctx *Context, name, data, size, provider string) (NanosVolume, error) {
	return CreateLocalVolume(ctx.config, name, data, size, provider)
}

// GetAllVolumes prints list of all onprem nanos-managed volumes
func (op *OnPrem) GetAllVolumes(ctx *Context) (*[]NanosVolume, error) {
	vols, err := GetVolumes(ctx.config.BuildDir, nil)
	if err != nil {
		return nil, err
	}

	return &vols, nil
}

// DeleteVolume deletes nanos-managed volume (filename and symlink)
func (op *OnPrem) DeleteVolume(ctx *Context, name string) error {
	query := map[string]string{
		"label": name,
		"id":    name,
	}

	buildDir := ctx.config.BuildDir

	volumes, err := GetVolumes(buildDir, query)
	if err != nil {
		return err
	}

	if len(volumes) == 1 {
		volumePath := path.Join(volumes[0].Path)
		err := os.Remove(volumePath)
		if err != nil {
			return err
		}
		symlinkPath := path.Join(buildDir, volumes[0].Name+".raw")
		err = os.Remove(symlinkPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// AttachVolume attaches volume to instance on `ops instance create -t onprem`
// or `ops run --mounts`
// on `ops image create --mount`, it simply creates a mount path
// with the given volume label
// label can refer to volume UUID or volume label
func (op *OnPrem) AttachVolume(ctx *Context, image, name, mount string) error {
	fmt.Println("not implemented")
	fmt.Println("use <ops run> or <ops load> with --mounts flag instead")
	fmt.Println("alternatively, use <ops image create -t onprem> with --mounts flag")
	fmt.Println("and run it with <ops instance create -t onprem>")
	return nil
}

// DetachVolume detaches volume
func (op *OnPrem) DetachVolume(ctx *Context, image, name string) error {
	fmt.Println("not implemented")
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

// buildVolumeManifest builds manifests for non-empty volume
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

// cleanUpVolumeManifest cleans up manifests for non-empty volume
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
	return nil
}

// GetVolumes get nanos volume using filter
// TODO might be better to interface this
func GetVolumes(dir string, query map[string]string) ([]NanosVolume, error) {
	var vols []NanosVolume
	mvols := make(map[string]NanosVolume)

	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		return vols, err
	}

	// this scans the directory twice, which can be improved
	// looking for symlink first
	for _, info := range fi {
		if info.IsDir() {
			continue
		}

		link, err := os.Readlink(path.Join(dir, info.Name()))
		if err != nil {
			continue
		}

		var id string
		var label string
		nl := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), VolumeDelimiter)
		if len(nl) == 1 {
			label = nl[0]
		}
		src, err := os.Stat(link)
		// ignore dangling symlink
		if err != nil {
			continue
		}
		nu := strings.Split(strings.TrimSuffix(src.Name(), ".raw"), VolumeDelimiter)
		if len(nu) == 2 {
			id = nu[1]
		}

		mvols[src.Name()] = NanosVolume{
			ID:        id,
			Name:      label,
			Label:     label,
			Size:      bytes2Human(src.Size()),
			Path:      path.Join(dir, src.Name()),
			CreatedAt: src.ModTime().String(),
		}
	}
	for _, info := range fi {
		if info.IsDir() {
			continue
		}

		link, _ := os.Readlink(path.Join(dir, info.Name()))
		if link != "" {
			continue
		}

		_, ok := mvols[info.Name()]
		if ok {
			continue
		}

		var id string
		nu := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), VolumeDelimiter)
		if len(nu) == 2 {
			id = nu[1]
		}
		mvols[info.Name()] = NanosVolume{
			ID:   id,
			Name: nu[0],
			Size: bytes2Human(info.Size()),
			Path: path.Join(dir, info.Name()),
		}
	}

	for _, vol := range mvols {
		vols = append(vols, vol)
	}
	if query == nil {
		return vols, nil
	}

	vols = filterVolume(vols, query)
	return vols, nil
}

// emulate kv store to easily extend query terms
// although multiple marshals/unmarshals makes this too convoluted
// another approach is to simply filter by designated field and repeat
// each time we need to query another field
func filterVolume(all []NanosVolume, query map[string]string) []NanosVolume {
	var vols []NanosVolume
	b, _ := json.Marshal(all)
	// empty slice and repopulate with filtered results
	all = nil
	var tmpVols []map[string]interface{}
	json.Unmarshal(b, &tmpVols)
	for k, v := range query {
		var vol NanosVolume
		for _, tmp := range tmpVols {
			// check if key is queried
			vv, ok := tmp[k]
			if !ok {
				continue
			}

			if vv == v {
				b, _ := json.Marshal(tmp)
				json.Unmarshal(b, &vol)
				vols = append(vols, vol)
			}
		}
	}
	return vols
}

// AddMounts adds Mounts and RunConfig.Mounts to image from flags
func AddMounts(mounts []string, config *Config) error {
	if config.Mounts == nil {
		config.Mounts = make(map[string]string)
	}
	query := make(map[string]string)

	for _, mnt := range mounts {
		lm := strings.Split(mnt, VolumeDelimiter)
		if len(lm) != 2 {
			return fmt.Errorf("mount config invalid: missing parts: %s", mnt)
		}
		if lm[1] == "" || lm[1][0] != '/' {
			return fmt.Errorf("mount config invalid: %s", mnt)
		}

		query["id"] = lm[0]
		query["label"] = lm[0]
		vols, err := GetVolumes(config.BuildDir, query)
		if err != nil {
			return err
		}

		if len(vols) == 0 {
			return fmt.Errorf("volume with uuid/label %s not found", lm[0])
		} else if len(vols) > 1 {
			return fmt.Errorf("ambiguous volume uuid/label: %s: multiple volumes found", lm[0])
		}
		_, ok := config.Mounts[lm[0]]
		if ok {
			return fmt.Errorf("mount path occupied: %s", lm[0])
		}
		config.Mounts[lm[0]] = lm[1]
		config.RunConfig.Mounts = append(config.RunConfig.Mounts, vols[0].Path)
	}

	return nil
}

// addMounts adds RunConfig.Mounts to image from existing Mounts
// to simulate attach/detach volume locally
func addMounts(config *Config) error {
	if config.Mounts == nil {
		return fmt.Errorf("no mount configuration found for image")
	}
	query := make(map[string]string)

	for label := range config.Mounts {
		query["id"] = label
		query["label"] = label
		vols, err := GetVolumes(config.BuildDir, query)
		if err != nil {
			return err
		}

		if len(vols) == 0 {
			return fmt.Errorf("volume with uuid/label %s not found", label)
		} else if len(vols) > 1 {
			return fmt.Errorf("ambiguous volume uuid/label: %s: multiple volumes found", label)
		}
		config.RunConfig.Mounts = append(config.RunConfig.Mounts, vols[0].Path)
	}

	return nil
}
