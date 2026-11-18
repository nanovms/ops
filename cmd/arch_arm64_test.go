package cmd

import "testing"

func TestGetPreferredArch(t *testing.T) {
	parch := getPreferredArch("amd64")
	if parch != "amd64" {
		t.Fatal("wrong preferred arch")
	}
}
