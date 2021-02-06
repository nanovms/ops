package fs

import (
	"testing"
)

func CheckMKFSSize(t *testing.T, mkfs *MkfsCommand, s string, size int64) {
	err := mkfs.SetFileSystemSize(s)
	if err != nil {
		t.Errorf("size %s: got error %v", s, err)
	} else if mkfs.size != size {
		t.Errorf("size: got %d want %d", mkfs.size, size)
	}
}

func TestMKFSCommand(t *testing.T) {
	mkfs := NewMkfsCommand(nil)

	t.Run("filesystem size", func(t *testing.T) {
		err := mkfs.SetFileSystemSize("size")
		if err == nil {
			t.Errorf("invalid size 'size': got %d want error", mkfs.size)
		}
		CheckMKFSSize(t, mkfs, "12", sectorSize)
		CheckMKFSSize(t, mkfs, "1k", 1024)
		CheckMKFSSize(t, mkfs, "1K", 1024)
		CheckMKFSSize(t, mkfs, "16K", 16384)
		CheckMKFSSize(t, mkfs, "2M", 2048*1024)
		CheckMKFSSize(t, mkfs, "4G", 4096*1024*1024)
		err = mkfs.SetFileSystemSize("2s")
		if err == nil {
			t.Errorf("invalid size '2s': got %d want error", mkfs.size)
		}
	})

	t.Run("execute without output file path", func(t *testing.T) {
		err := mkfs.Execute()
		if err == nil {
			t.Errorf("nil error")
		}
	})
}
