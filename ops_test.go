package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func TestDownloadImages(t *testing.T) {
	// remove the files to force a download
	// ignore any error from remove
	os.Remove(api.Mkfs)
	os.Remove(api.BootImg)
	os.Remove(api.KernelImg)
	api.DownloadBootImages()

	if _, err := os.Stat(api.BootImg); os.IsNotExist(err) {
		t.Errorf(".staging/boot file not found")
	}

	if info, err := os.Stat(api.Mkfs); os.IsNotExist(err) {
		t.Errorf("mkfs not found")
	} else {
		mode := fmt.Sprintf("%04o", info.Mode().Perm())
		if mode != "0775" {
			t.Errorf("mkfs not executable")
		}
	}

	if _, err := os.Stat(api.KernelImg); os.IsNotExist(err) {
		t.Errorf(".staging/stage3 file not found")
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
	err := api.BuildImage(userImage, c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := api.HypervisorInstance()
	go func() {
		hypervisor.Start(api.FinalImg, 8080)
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
	api.DownloadBootImages()
	var c api.Config
	c.Dirs = []string{"data/static"}
	err := api.BuildImage("data/main", c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := api.HypervisorInstance()
	go func() {
		hypervisor.Start(api.FinalImg, 8080)
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
	api.DownloadBootImages()
	runHyperVisor("./data/webg", "unibooty 0", t)
}

func TestStartHypervisor(t *testing.T) {
	api.DownloadBootImages()
	runHyperVisor("./data/webs", "unibooty!", t)
}
