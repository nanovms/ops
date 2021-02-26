package onprem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/types"
)

const (
	// MinimumVolumeSize is the minimum size of a volume created with mkfs (1 MB).
	MinimumVolumeSize = MByte
)

// CreateVolume creates volume for onprem image
func (op *OnPrem) CreateVolume(ctx *lepton.Context, name, data, size, provider string) (lepton.NanosVolume, error) {
	return lepton.CreateLocalVolume(ctx.Config(), name, data, size, provider)
}

// GetAllVolumes prints list of all onprem nanos-managed volumes
func (op *OnPrem) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	vols, err := GetVolumes(ctx.Config().BuildDir, nil)
	if err != nil {
		return nil, err
	}

	return &vols, nil
}

// DeleteVolume deletes nanos-managed volume (filename and symlink)
func (op *OnPrem) DeleteVolume(ctx *lepton.Context, name string) error {
	query := map[string]string{
		"label": name,
		"id":    name,
	}

	buildDir := ctx.Config().BuildDir

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
func (op *OnPrem) AttachVolume(ctx *lepton.Context, image, name, mount string) error {
	fmt.Println("not implemented")
	fmt.Println("use <ops run> or <ops load> with --mounts flag instead")
	fmt.Println("alternatively, use <ops image create -t onprem> with --mounts flag")
	fmt.Println("and run it with <ops instance create -t onprem>")
	return nil
}

// DetachVolume detaches volume
func (op *OnPrem) DetachVolume(ctx *lepton.Context, image, name string) error {
	fmt.Println("not implemented")
	return nil
}

// parseSize parses the size of the lepton.NanosVolume to human readable format.
// If the size value is empty, it returns 1 MB (the default size of volumes).
func (op *OnPrem) parseSize(vol lepton.NanosVolume) string {
	if vol.Size == "" {
		// return the default size of a volume
		return lepton.Bytes2Human(MiByte)
	}
	bytes, err := parseBytes(vol.Size)
	if err != nil {
		fmt.Printf("warning: invalid size value for volume %s with UUID %s: %s\n", vol.Name, vol.ID, err.Error())
	}
	size := lepton.Bytes2Human(bytes)
	return size
}

// GetVolumes get nanos volume using filter
// TODO might be better to interface this
func GetVolumes(dir string, query map[string]string) ([]lepton.NanosVolume, error) {
	var vols []lepton.NanosVolume
	mvols := make(map[string]lepton.NanosVolume)

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
		nl := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), lepton.VolumeDelimiter)
		if len(nl) == 1 {
			label = nl[0]
		}
		src, err := os.Stat(link)
		// ignore dangling symlink
		if err != nil {
			continue
		}
		nu := strings.Split(strings.TrimSuffix(src.Name(), ".raw"), lepton.VolumeDelimiter)
		if len(nu) == 2 {
			id = nu[1]
		}

		mvols[src.Name()] = lepton.NanosVolume{
			ID:        id,
			Name:      label,
			Label:     label,
			Size:      lepton.Bytes2Human(src.Size()),
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
		nu := strings.Split(strings.TrimSuffix(info.Name(), ".raw"), lepton.VolumeDelimiter)
		if len(nu) == 2 {
			id = nu[1]
		}
		mvols[info.Name()] = lepton.NanosVolume{
			ID:   id,
			Name: nu[0],
			Size: lepton.Bytes2Human(info.Size()),
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
func filterVolume(all []lepton.NanosVolume, query map[string]string) []lepton.NanosVolume {
	var vols []lepton.NanosVolume
	b, _ := json.Marshal(all)
	// empty slice and repopulate with filtered results
	all = nil
	var tmpVols []map[string]interface{}
	json.Unmarshal(b, &tmpVols)
	for k, v := range query {
		var vol lepton.NanosVolume
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
func AddMounts(mounts []string, config *types.Config) error {
	if config.Mounts == nil {
		config.Mounts = make(map[string]string)
	}
	query := make(map[string]string)

	for _, mnt := range mounts {
		lm := strings.Split(mnt, lepton.VolumeDelimiter)
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

// AddMountsFromConfig adds RunConfig.Mounts to image from existing Mounts
// to simulate attach/detach volume locally
func AddMountsFromConfig(config *types.Config) error {
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
