package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func TestDownloadImages(t *testing.T) {

	home, err := api.HomeDir()
	if err != nil {
		t.Errorf("Failed to get HOME directory")
	}
	mkfsFilePath := path.Join(home, api.OpsDir, api.Mkfs)
	bootImgFilePath := path.Join(home, api.OpsDir, api.BootImg)
	kernelImgFilePath := path.Join(home, api.OpsDir, api.KernelImg)

	// remove the files to force a download
	// ignore any error from remove
	os.Remove(mkfsFilePath)
	os.Remove(bootImgFilePath)
	os.Remove(kernelImgFilePath)

	api.DownloadBootImages("", false)

	if _, err := os.Stat(bootImgFilePath); os.IsNotExist(err) {
		t.Errorf("%s/%s/boot file not found", home, api.OpsDir)
	}

	if info, err := os.Stat(mkfsFilePath); os.IsNotExist(err) {
		t.Errorf("%s/%s/mkfs not found", home, api.OpsDir)
	} else {
		mode := fmt.Sprintf("%04o", info.Mode().Perm())
		if mode != "0775" {
			t.Errorf("%s/%s/mkfs not executable", home, api.OpsDir)
		}
	}

	if _, err := os.Stat(kernelImgFilePath); os.IsNotExist(err) {
		t.Errorf("%s/%s/stage3 file not found", home, api.OpsDir)
	}
}

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOutput(buf)
	root.SetArgs(args)
	fmt.Println(args)
	c, err = root.ExecuteC()
	return c, buf.String(), err
}

func runHyperVisor(userImage string, expected string, t *testing.T) {
	var c api.Config
	c.Program = userImage
	err := api.BuildImage(c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := api.HypervisorInstance()
	rconfig := api.RuntimeConfig(api.FinalImg, []int{8080}, true)
	go func() {
		hypervisor.Start(&rconfig)
	}()
	time.Sleep(3 * time.Second)
	resp, err := http.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Log(err)
		t.Errorf("failed to get 127.0.0.1:8080")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log(err)
		t.Errorf("ReadAll failed")
	}

	if string(body) != expected {
		t.Errorf("unexpected response" + string(body))
	}
	hypervisor.Stop()
}

func TestImageWithStaticFiles(t *testing.T) {
	api.DownloadBootImages("", false)
	var c api.Config
	c.Dirs = []string{"data/static"}
	c.Program = "data/main"
	err := api.BuildImage(c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := api.HypervisorInstance()
	rconfig := api.RuntimeConfig(api.FinalImg, []int{8080}, true)
	go func() {
		hypervisor.Start(&rconfig)
	}()
	time.Sleep(3 * time.Second)
	resp, err := http.Get("http://localhost:8080/example.html")
	if err != nil {
		t.Log(err)
		t.Errorf("failed to get :8080/example.html")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log(err)
		t.Errorf("ReadAll failed")
	}
	fmt.Println(string(body))
	hypervisor.Stop()
}

func TestRunningDynamicImage(t *testing.T) {
	api.DownloadBootImages("", false)
	runHyperVisor("./data/webg", "unibooty 0", t)
}

func TestStartHypervisor(t *testing.T) {
	api.DownloadBootImages("", false)
	runHyperVisor("./data/webs", "unibooty!", t)
}
