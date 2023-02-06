//go:build linux
// +build linux

package lepton

import (
	"os"
	"testing"
)

func TestGetSharedLibsSystemLs(t *testing.T) {
	if _, err := os.Stat("/bin/ls"); err != nil {
		t.Skip("could not stat /bin/ls:", err)
	}
	c := NewConfig()
	targetRoot := os.Getenv("NANOS_TARGET_ROOT")
	libs, err := getSharedLibs(targetRoot, "/bin/ls", c)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range libs {
		t.Logf("%s -> %s", k, v)
	}
}
