package onprem

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

const (
	// MinimumVolumeSize is the minimum size of a volume created with mkfs (1 MB).
	MinimumVolumeSize = MByte
)

// CreateVolume creates volume for onprem image
func (op *OnPrem) CreateVolume(ctx *lepton.Context, name, data, provider string) (lepton.NanosVolume, error) {
	c := ctx.Config()
	if c.BaseVolumeSz == "" {
		c.BaseVolumeSz = strconv.Itoa(MinimumVolumeSize)
	}
	return lepton.CreateLocalVolume(c, name, data, provider)
}

// GetAllVolumes prints list of all onprem nanos-managed volumes
func (op *OnPrem) GetAllVolumes(ctx *lepton.Context) (*[]lepton.NanosVolume, error) {
	vols, err := GetVolumes(ctx.Config().VolumesDir, nil)
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

	buildDir := ctx.Config().VolumesDir
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
	}

	return nil
}

// AttachVolume attaches volume to instance on `ops instance create -t onprem`
// or `ops run --mounts`
// on `ops image create --mount`, it simply creates a mount path
// with the given volume label
// label can refer to volume UUID or volume label
//
//	You must start the instance with QMP otherwise this won't work.
//	{
//	  "RunConfig": {
//	    "QMP": true
//	  }
//	}
//
// this currently requires instance name to be unique
func (op *OnPrem) AttachVolume(ctx *lepton.Context, instanceName string, volumeName string, attachID int) error {
	vol := ""

	buildDir := ctx.Config().VolumesDir
	vols, err := GetVolumes(buildDir, nil)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(vols); i++ {
		if vols[i].Name == volumeName {
			vol = vols[i].Path
		}
	}

	instance, err := op.GetMetaInstanceByName(ctx, instanceName)
	if err != nil {
		return err
	}

	last := instance.Mgmt

	deviceAddCmd := `{ "execute": "device_add", "arguments": {"driver": "scsi-hd", "bus": "scsi0.0", "drive": "` + volumeName + `", "id": "` + volumeName + `"`
	if attachID >= 0 {
		deviceAddCmd += fmt.Sprintf(`, "device_id": "persistent-disk-%d"`, attachID)
	}
	deviceAddCmd += `}}`
	commands := []string{
		`{ "execute": "qmp_capabilities" }`,
		`{ "execute": "blockdev-add", "arguments": {"driver": "raw", "node-name":"` + volumeName + `", "file": {"driver": "file", "filename": "` + vol + `"} } }`,
		deviceAddCmd,
	}

	executeQMP(commands, last)

	return nil
}

// DetachVolume detaches volume
func (op *OnPrem) DetachVolume(ctx *lepton.Context, instanceName string, volumeName string) error {
	vol := ""

	buildDir := ctx.Config().VolumesDir
	vols, err := GetVolumes(buildDir, nil)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(vols); i++ {
		if vols[i].Name == volumeName {
			vol = vols[i].Path
		}
	}

	if vol != "" {
		fmt.Printf("removing %s\n", vol)
	}

	instance, err := op.GetMetaInstanceByName(ctx, instanceName)
	if err != nil {
		return err
	}

	last := instance.Mgmt

	commands := []string{
		`{ "execute": "qmp_capabilities" }`,
		`{ "execute": "device_del", "arguments": {"id": "` + volumeName + `"}}`,
		`{ "execute": "blockdev-del", "arguments": {"node-name":"` + volumeName + `" } }`,
	}

	c, err := net.Dial("tcp", "localhost:"+last)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	for i := 0; i < len(commands); i++ {
		_, err := c.Write([]byte(commands[i] + "\n"))
		if err != nil {
			fmt.Println(err)
		}
		received := make([]byte, 1024)
		_, err = c.Read(received)
		if err != nil {
			println("Read data failed:", err.Error())
			os.Exit(1)
		}
	}

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

	files, err := os.ReadDir(dir)
	if err != nil {
		return vols, err
	}

	for _, info := range files {
		if info.IsDir() {
			continue
		}

		filename := strings.TrimSuffix(info.Name(), ".raw")
		nameParts := strings.Split(filename, lepton.VolumeDelimiter)
		if len(nameParts) < 2 { // invalid file name
			continue
		}

		fi, err := info.Info()
		if err != nil {
			return nil, err
		}

		vols = append(vols, lepton.NanosVolume{
			ID:        nameParts[1],
			Name:      nameParts[0],
			Label:     nameParts[0],
			Size:      lepton.Bytes2Human(fi.Size()),
			Path:      path.Join(dir, info.Name()),
			CreatedAt: fi.ModTime().String(),
		})
	}

	return filterVolume(vols, query)
}

// Filter given volumes to match given query
func filterVolume(all []lepton.NanosVolume, query map[string]string) ([]lepton.NanosVolume, error) {
	if len(query) == 0 {
		return all, nil
	}

	var vols []lepton.NanosVolume
	for _, vol := range all {
		if vol.MatchedByQueries(query) {
			vols = append(vols, vol)
		}
	}
	return vols, nil
}

// AddVirtfsShares sets up RunConfig.VirtfsShares for the hypervisor
func AddVirtfsShares(config *types.Config) error {
	query := make(map[string]string)
	virtfsShares := make(map[string]string)

	for label, mountDir := range config.Mounts {
		query["id"] = label
		query["label"] = label
		vols, err := GetVolumes(lepton.LocalVolumeDir, query)
		if err != nil {
			return err
		}
		if len(vols) == 0 {
			// There are no local volumes matching this mount directive: look for a matching local folder
			var hostDir string
			if path.IsAbs(label) {
				hostDir = label
			} else {
				hostDir = path.Join(config.LocalFilesParentDirectory, label)
			}
			info, err := os.Stat(hostDir)
			if err != nil {
				return err
			}
			if info.IsDir() {
				log.Info("Adding virtFS share", hostDir)
				virtfsShares[hostDir] = mountDir
				delete(config.Mounts, label) // This mount entry is replaced with an entry containing a virtual ID
			}
		}
	}
	for hostDir, mountDir := range virtfsShares {
		config.RunConfig.VirtfsShares = append(config.RunConfig.VirtfsShares, hostDir)
		config.Mounts[fmt.Sprintf("%%%d", len(config.RunConfig.VirtfsShares))] = mountDir
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
		if label[0] == '%' { // virtual ID
			continue
		}
		query["id"] = label
		query["label"] = label
		vols, err := GetVolumes(config.VolumesDir, query)
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

// CreateVolumeImage ...
func (op *OnPrem) CreateVolumeImage(ctx *lepton.Context, imageName, data, provider string) (lepton.NanosVolume, error) {
	return lepton.NanosVolume{}, errors.New("Unsupported")
}

// CreateVolumeFromSource ...
func (op *OnPrem) CreateVolumeFromSource(ctx *lepton.Context, sourceType, sourceName, volumeName, provider string) error {
	return errors.New("Unsupported")
}
