package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nanovms/ops/lepton"
	"github.com/stretchr/testify/assert"
)

func captureStdout(f func()) string {
	ostdout := os.Stdout

	r, w, _ := os.Pipe()

	os.Stdout = w

	f()

	w.Close()

	os.Stdout = ostdout

	fout, _ := io.ReadAll(r)
	r.Close()

	return string(fout)
}

func TestListPkgCommand(t *testing.T) {

	rt := getPkgArch()

	fbody := `{
	   "Version": 2,
	   "Packages": [
	   {
	   "description": "Apache httpd web server",
	   "language": "c",
	   "name": "apache",
	   "sha256": "1723e7ce2a9a3475559089ee1e6e440d22b63c8b9e630b61daf2f5aedbb378e4",
	   "download_count": 28,
	   "size": 3580,
	   "version": "2.4.48",
	   "namespace": "eyberg",
	   "readme": "# Apache 2.4.48 web server package for the Nanos unikernel",
	   "arch": "` + rt + `",
	   "created_at": "2022-02-16T21:02:15Z"
	   },
	   {
	   "description": "in memory k/v store",
	   "language": "c",
	   "name": "redis",
	   "sha256": "8e161ef2ba1bca4f603e77e03df7cb058691d81cb25002b853c7965a3a47cc2e",
	   "download_count": 141,
	   "size": 5115,
	   "version": "5.0.5",
	   "namespace": "eyberg",
	   "readme": "",
		   "arch": "` + rt + `",
	   "created_at": "2022-02-16T21:02:23Z"
	   }
]
}
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, fbody)
	}))

	defer server.Close()

	lepton.PackageManifestURL = server.URL

	rcmd := GetRootCommand()

	rcmd.SetArgs([]string{"pkg", "list", "--json=true"})

	output := captureStdout(func() {
		err := rcmd.Execute()
		if err != nil {
			t.Fatal(err)
		}
	})

	var pl []lepton.Package

	err := json.Unmarshal([]byte(output), &pl)
	if err != nil {
		fmt.Println(err)
	}

	if len(pl) != 2 {
		t.Fatal(err)
	}

}

func TestGetPkgCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"get", "eyberg/bind:9.13.4", "--arch", "amd64"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgContentsCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"contents", "eyberg/bind:9.13.4"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestPkgDescribeCommand(t *testing.T) {
	getPkgCmd := PackageCommands()

	getPkgCmd.SetArgs([]string{"describe", "eyberg/bind:9.13.4", "--arch", "amd64"})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)
}

func TestLoad(t *testing.T) {

	getPkgCmd := PackageCommands()

	program := buildNodejsProgram()
	defer os.Remove(program)

	getPkgCmd.SetArgs([]string{"load", "eyberg/node:v14.2.0", "--arch", "amd64", "-a", program})

	err := getPkgCmd.Execute()

	assert.Nil(t, err)

	removeImage(program)
}

func TestPkgSearch(t *testing.T) {
	searchPkgCmd := PackageCommands()

	searchPkgCmd.SetArgs([]string{"search", "mysql"})
	err := searchPkgCmd.Execute()
	assert.Nil(t, err)
}
