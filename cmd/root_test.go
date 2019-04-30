package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

const START_WAIT_TIMEOUT = time.Second * 30

func runAndWaitForString(rconfig *api.RunConfig, timeout time.Duration, text string, t *testing.T) api.Hypervisor {
	hypervisor := api.HypervisorInstance()
	if hypervisor == nil {
		t.Fatal("No hypervisor found on $PATH")
	}
	cmd := hypervisor.Command(rconfig)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	reader := io.TeeReader(stdoutPipe, os.Stdout)
	cmd.Stderr = os.Stderr
	cmd.Start()

	done := make(chan struct{})
	errch := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), text) {
				done <- struct{}{}
			}
		}
		if err := scanner.Err(); err != nil {
			errch <- err
		}
		errch <- errors.New("Expected text not found")
	}()

	select {
	case <-time.After(timeout):
		hypervisor.Stop()
		t.Fatal("Timed out")
	case err := <-errch:
		hypervisor.Stop()
		t.Fatal(err)
	case <-done:
		break
	}
	return hypervisor
}
func TestDownloadImages(t *testing.T) {
	// remove the files to force a download
	// ignore any error from remove
	boot := path.Join(api.GetOpsHome(), api.LatestReleaseVersion, "boot.img")
	kernel := path.Join(api.GetOpsHome(), api.LatestReleaseVersion, "stage3.img")
	mkfs := path.Join(api.GetOpsHome(), api.LatestReleaseVersion, "mkfs")
	os.Remove(mkfs)
	os.Remove(boot)
	os.Remove(kernel)
	api.DownloadReleaseImages(api.LatestReleaseVersion)

	if _, err := os.Stat(boot); os.IsNotExist(err) {
		t.Errorf("%v file not found", boot)
	}

	if info, err := os.Stat(mkfs); os.IsNotExist(err) {
		t.Errorf("%v not found", mkfs)
	} else {
		if (info.Mode().Perm() & 0111) == 0 {
			t.Errorf("mkfs not executable")
		}
	}

	if _, err := os.Stat(kernel); os.IsNotExist(err) {
		t.Errorf("%v not found", kernel)
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

func runHyperVisor(userImage string, started string, expected string, t *testing.T) {
	var c api.Config
	c.Program = userImage
	c.TargetRoot = os.Getenv("NANOS_TARGET_ROOT")
	c.RunConfig = api.RuntimeConfig(api.GenerateImageName(c.Program), []int{8080}, true)
	fixupConfigImages(&c, api.LocalReleaseVersion)
	err := api.BuildImage(c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := runAndWaitForString(&c.RunConfig, START_WAIT_TIMEOUT, started, t)
	defer hypervisor.Stop()

	resp, err := http.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Log(err)
		t.Fatalf("failed to get 127.0.0.1:8080")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log(err)
		t.Fatalf("ReadAll failed")
	}

	if string(body) != expected {
		t.Errorf("unexpected response" + string(body))
	}
}

func TestImageWithStaticFiles(t *testing.T) {
	api.DownloadReleaseImages(api.LatestReleaseVersion)
	var c api.Config
	c.Dirs = []string{"../data/static"}
	c.Program = "../data/main"
	c.TargetRoot = os.Getenv("NANOS_TARGET_ROOT")
	c.RunConfig = api.RuntimeConfig(api.GenerateImageName(c.Program), []int{8080}, true)
	fixupConfigImages(&c, api.LatestReleaseVersion)
	err := api.BuildImage(c)
	if err != nil {
		t.Fatal(err)
	}
	hypervisor := runAndWaitForString(&c.RunConfig, START_WAIT_TIMEOUT, "Listening...", t)
	defer hypervisor.Stop()

	resp, err := http.Get("http://localhost:8080/example.html")
	if err != nil {
		t.Log(err)
		t.Fatalf("failed to get :8080/example.html")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Log(err)
		t.Fatalf("ReadAll failed")
	}

	fmt.Println(string(body))
}

func TestRunningDynamicImage(t *testing.T) {
	api.DownloadReleaseImages(api.LatestReleaseVersion)
	runHyperVisor("../data/webg", "Server started", "unibooty 0", t)
}

func TestStartHypervisor(t *testing.T) {
	api.DownloadReleaseImages(api.LatestReleaseVersion)
	runHyperVisor("../data/webs", "Server started", "unibooty!", t)
}
