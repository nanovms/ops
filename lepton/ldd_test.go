package lepton

import (
	"os"
	"testing"
)

func TestGetSharedLibs(t *testing.T) {
	targetRoot := os.Getenv("NANOS_TARGET_ROOT")
	deps, err := getSharedLibs(targetRoot, "../data/webg")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) == 0 {
		t.Fatal("No deps for ../data/webg")
	}
}

func TestGetSharedLibsSystemLs(t *testing.T) {
	if _, err := os.Stat("/bin/ls"); err != nil {
		t.Skip("could not stat /bin/ls:", err)
	}
	targetRoot := os.Getenv("NANOS_TARGET_ROOT")
	if _, err := getSharedLibs(targetRoot, "/bin/ls"); err != nil {
		t.Fatal(err)
	}
}
