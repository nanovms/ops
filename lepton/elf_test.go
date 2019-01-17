package lepton

import (
	"testing"
)

func TestGetSharedLibs(t *testing.T) {
	deps, err := getSharedLibs("../data/webg")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) == 0 {
		t.Fatal("No deps for ../data/webg")
	}
}
