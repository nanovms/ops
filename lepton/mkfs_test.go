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
		mkfs.SetEmptyFileSystem()
		mkfs.SetLabel("label")

		got := mkfs.GetArgs()
		want := []string{
			"rawPath",
			"-s",
			"size",
			"-r",
			"targetRoot",
			"-b",
			"boot",
			"-e",
			"-l",
			"label",
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v want %v", got, want)
		}
	})
}
