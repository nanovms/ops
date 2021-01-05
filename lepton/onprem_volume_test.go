package lepton

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var (
	testVolumeConfig = &Config{
		Mkfs: path.Join(GetOpsHome(), "nightly", "mkfs"),
	}
	testVolume1 = &NanosVolume{
		ID:    "",
		Name:  "empty",
		Label: "empty",
		Data:  "",
		Size:  "",
		Path:  "",
	}
	testVolume2 = &NanosVolume{
		ID:    "",
		Name:  "noempty",
		Label: "noempty",
		Data:  "data",
		Size:  "",
		Path:  "",
	}
	testOP = &OnPrem{}
)

func NewTestContext(c *Config) *Context {
	return &Context{
		config: testVolumeConfig,
		logger: NewLogger(ioutil.Discard),
	}
}

func TestOnPremVolume(t *testing.T) {
	// set up here since linter/vet complains about using testing.Main
	tmp, err := ioutil.TempDir("/tmp", "test-ops-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	tmpdata, err := ioutil.TempDir(tmp, testVolume2.Data)
	if err != nil {
		t.Fatal(err)
	}
	DownloadNightlyImages(testVolumeConfig)

	testVolumeConfig.BuildDir = tmp
	testVolume2.Data = tmpdata
	count := new(int)
	*count = 0

	testGetVolumes(t, "get_volumes_0", count)
	testCreateVolume(t, "volume_1", testVolume1, count)

	testCreateVolume(t, "volume_2", testVolume2, count)
	testDeleteVolumeByName(t, "volume_1", testVolume1, count)
	testDeleteVolumeByUUID(t, "volume_2", testVolume2, count)
}

func testCreateVolume(t *testing.T, name string, vol *NanosVolume, count *int) {
	t.Run(fmt.Sprintf("create_%s", name), func(t *testing.T) {
		res, err := testOP.CreateVolume(NewTestContext(testVolumeConfig), vol.Name, vol.Data, vol.Size, "onprem")
		if err != nil {
			t.Error(err)
			return
		}
		*count++
		assignVolumeData(res, vol)
		// only test GetVolumes if create is successful
		testGetVolumes(t, fmt.Sprintf("get_after_create_%s", name), count)
	})
}

func testGetVolumes(t *testing.T, name string, count *int) {
	t.Run(name, func(t *testing.T) {
		vols, err := GetVolumes(testVolumeConfig.BuildDir, nil)
		if err != nil {
			t.Error(err)
			return
		}
		if len(vols) != *count {
			t.Errorf("expected %d, got %d", *count, len(vols))
		}
	})
}

func testDeleteVolumeByName(t *testing.T, name string, vol *NanosVolume, count *int) {
	t.Run(fmt.Sprintf("delete_by_name_%s", name), func(t *testing.T) {
		err := testOP.DeleteVolume(NewTestContext(testVolumeConfig), vol.Name)
		if err != nil {
			t.Error(err)
			return
		}
		*count--
		// only test GetVolumes if delete is successful
		testGetVolumes(t, fmt.Sprintf("get_after_delete_%s", name), count)
	})
}

func testDeleteVolumeByUUID(t *testing.T, name string, vol *NanosVolume, count *int) {
	t.Run(fmt.Sprintf("delete_by_name_%s", name), func(t *testing.T) {
		err := testOP.DeleteVolume(NewTestContext(testVolumeConfig), vol.ID)
		if err != nil {
			t.Error(err)
			return
		}
		*count--
		// only test GetVolumes if delete is successful
		testGetVolumes(t, fmt.Sprintf("get_after_delete_%s", name), count)
	})
}

func assignVolumeData(src NanosVolume, dst *NanosVolume) {
	dst.ID = src.ID
	dst.Name = src.Name
	dst.Label = src.Label
	dst.Data = src.Data
	dst.Path = src.Path
}

func TestOnPremVolume_AddMounts(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "testOPs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	config := &Config{BuildDir: dir}

	tests := []struct {
		title   string
		name    string
		uuid    string
		label   string
		mount   string
		mountAt string
		err     bool
	}{
		{
			title:   "mount_by_uuid",
			name:    "empty",
			uuid:    "uuid-1",
			mount:   "uuid-1",
			mountAt: "/mnt",
		},
		{
			title: "mount_invalid_incomplete",
			mount: "uuid-1",
			err:   true,
		},
		{
			title:   "mount_invalid_no_slash",
			mount:   "uuid-1",
			mountAt: "mnt",
			err:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var mounts []string
			err := ioutil.WriteFile(path.Join(dir, fmt.Sprintf("%s:%s.raw", tt.name, tt.uuid)), []byte{}, 0644)
			if err != nil {
				t.Errorf("TempFile: %v", err)
				return
			}
			mounts = append(mounts, fmt.Sprintf("%s:%s", tt.mount, tt.mountAt))
			err = AddMounts(mounts, config)
			if err != nil {
				if !tt.err {
					t.Errorf("AddMounts: %v", err)
				}
				return
			}
			err = addMounts(config)
			if err != nil {
				if !tt.err {
					t.Errorf("addMounts: %v", err)
				}
				return
			}
		})
	}
}
