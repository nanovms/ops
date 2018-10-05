package lepton

import (
	"testing"
)

func TestGetSharedLibs(t *testing.T) {
	deps, err := getSharedLibs("/bin/bash")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) == 0 {
		t.Fatal("No deps for /bin/bash")
	}
}
