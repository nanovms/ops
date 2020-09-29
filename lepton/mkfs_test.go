package lepton

import (
	"reflect"
	"testing"
)

func TestMKFSCommand(t *testing.T) {
	mkfs := NewMkfsCommand("")

	t.Run("should test arguments functions", func(t *testing.T) {
		mkfs.SetFileSystemPath("rawPath")
		mkfs.SetFileSystemSize("size")
		mkfs.SetTargetRoot("targetRoot")
		mkfs.SetBoot("boot")
		mkfs.SetImageName("imageName")

		got := mkfs.GetArgs()
		want := []string{
			"-e",
			"rawPath",
			"-s",
			"size",
			"-r",
			"targetRoot",
			"-b",
			"boot",
			"imageName",
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Got %v want %v", got, want)
		}
	})
}
