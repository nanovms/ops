package fs

import (
	"testing"
)

func TestLogExt(t *testing.T) {
	tfs := &tfs{
		imgOffset: 0,
		size:      512,
	}
	t.Run("initial log ext", func(t *testing.T) {
		ext := tfs.newLogExt(true, false)
		if string(ext.buffer[:6]) != "NVMTFS" {
			t.Errorf("invalid TFS magic: got %d want 'NVMTFS'", ext.buffer[:6])
		}
	})
}
