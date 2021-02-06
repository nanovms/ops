package lepton

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// Version for ops
var Version string

// OpsReleaseURL gives URL to download latest ops binary
const OpsReleaseURL = "https://storage.googleapis.com/cli/%v/ops"

const releaseBaseURL string = "https://storage.googleapis.com/nanos/release/"

const nightlyReleaseBaseURL string = "https://storage.googleapis.com/nanos/release/nightly/"

// PackageBaseURL gives URL for downloading of packages
const PackageBaseURL string = "https://storage.googleapis.com/packagehub/%v"

// PackageManifestURL stores info about all packages
const PackageManifestURL string = "https://storage.googleapis.com/packagehub/manifest.json"

// PackageManifestFileName is manifest file path
const PackageManifestFileName string = "manifest.json"

// GCPStorageURL is GCP storage path
const GCPStorageURL string = "https://storage.googleapis.com/%v/%v"

// GenerateImageName generate image name
func GenerateImageName(program string) string {
	program = filepath.Base(program)
	images := path.Join(GetOpsHome(), "images")
	var buffer bytes.Buffer
	buffer.WriteString(images)
	buffer.WriteRune('/')
	buffer.WriteString(program)
	buffer.WriteString(".img")
	return buffer.String()
}

// PackagesCache where all packages are stored
var PackagesCache = getPackageCache()

// GetOpsHome get ops directory path
// We store all ops related info, packages, images in this directory
func GetOpsHome() string {
	home, err := HomeDir()
	if err != nil {
		panic(err)
	}

	opshome := path.Join(home, ".ops")
	images := path.Join(opshome, "images")
	instances := path.Join(opshome, "instances")
	manifests := path.Join(opshome, "manifests")
	volumes := path.Join(opshome, "volumes")

	if _, err := os.Stat(images); os.IsNotExist(err) {
		os.MkdirAll(images, 0755)
	}

	if _, err := os.Stat(instances); os.IsNotExist(err) {
		os.MkdirAll(instances, 0755)
	}

	if _, err := os.Stat(manifests); os.IsNotExist(err) {
		os.MkdirAll(manifests, 0755)
	}

	if _, err := os.Stat(volumes); os.IsNotExist(err) {
		os.MkdirAll(volumes, 0755)
	}

	return opshome
}

func getImageTempDir(c *Config) string {
	temp := filepath.Base(c.Program) + "_temp"

	if c.BuildDir == "" {
		dir, err := ioutil.TempDir("", temp)
		if err != nil {
			fmt.Println(err)
		}

		c.BuildDir = dir

	}

	return c.BuildDir
}

// NightlyReleaseURL give URL for nightly build
var NightlyReleaseURL = nightlyReleaseURL()

func nightlyFileName() string {
	return fmt.Sprintf("nanos-nightly-%v.tar.gz", runtime.GOOS)
}

func nightlyReleaseURL() string {
	var sb strings.Builder
	sb.WriteString(nightlyReleaseBaseURL)
	sb.WriteString(nightlyFileName())
	return sb.String()
}

func nightlyLocalFolder() string {
	return path.Join(GetOpsHome(), "nightly")
}

// NightlyLocalFolder is directory path where nightly builds are stored
var NightlyLocalFolder = nightlyLocalFolder()

// LocalTimeStamp gives local timestamp from download nightly build
func LocalTimeStamp() (string, error) {
	timestamp := fmt.Sprintf("nanos-nightly-%v.timestamp", runtime.GOOS)
	data, err := ioutil.ReadFile(path.Join(NightlyLocalFolder, timestamp))
	// first time download?
	if os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RemoteTimeStamp gives latest nightly build timestamp
func RemoteTimeStamp() (string, error) {
	timestamp := fmt.Sprintf("nanos-nightly-%v.timestamp", runtime.GOOS)
	resp, err := http.Get(nightlyReleaseBaseURL + timestamp)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func updateLocalTimestamp(timestamp string) error {
	fname := fmt.Sprintf("nanos-nightly-%v.timestamp", runtime.GOOS)
	return ioutil.WriteFile(path.Join(NightlyLocalFolder, fname), []byte(timestamp), 0755)
}

func updateLocalRelease(version string) error {
	local := path.Join(GetOpsHome(), "latest.txt")
	LocalReleaseVersion = version
	return ioutil.WriteFile(local, []byte(version), 0755)
}

// LatestReleaseVersion give latest stable release for nanos
var LatestReleaseVersion = getLatestRelVersion()

const (
	// WarningColor used in warning texts
	WarningColor = "\033[1;33m%s\033[0m"
	// ErrorColor used in error texts
	ErrorColor = "\033[1;31m%s\033[0m"
)

func getLatestRelVersion() string {
	resp, err := http.Get(releaseBaseURL + "latest.txt")
	if err != nil {
		fmt.Printf(WarningColor, "version lookup failed, using local.\n")
		if LocalReleaseVersion == "0.0" {
			fmt.Printf(ErrorColor, "No local build found.")
			os.Exit(1)
		}
		return LocalReleaseVersion
	}
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return strings.TrimSuffix(string(data), "\n")
}

// LocalReleaseVersion is version latest release downloaded in ops home
var LocalReleaseVersion = getLocalRelVersion()

func getLocalRelVersion() string {
	data, err := ioutil.ReadFile(path.Join(GetOpsHome(), "latest.txt"))
	// nothing yet, force a download
	if os.IsNotExist(err) {
		return "0.0"
	}
	return strings.TrimSuffix(string(data), "\n")
}

func releaseFileName(version string) string {
	return fmt.Sprintf("nanos-release-%v-%v.tar.gz", runtime.GOOS, version)
}

func getReleaseURL(version string) string {
	var sb strings.Builder
	sb.WriteString(releaseBaseURL)
	sb.WriteString(version)
	sb.WriteRune('/')
	sb.WriteString(releaseFileName(version))
	return sb.String()
}

func getReleaseLocalFolder(version string) string {
	return path.Join(GetOpsHome(), version)
}

func getLastReleaseLocalFolder() string {
	return getReleaseLocalFolder(getLatestRelVersion())
}

func getKlibsDir(nightly bool) string {
	if nightly {
		return nightlyLocalFolder() + "/klibs"
	}

	return getLastReleaseLocalFolder() + "/klibs"
}

const (
	commonArchive = "https://storage.googleapis.com/nanos/common/common.tar.gz"
	libDNS        = "/lib/x86_64-linux-gnu/libnss_dns.so.2"
	sslCERT       = "/etc/ssl/certs/ca-certificates.crt"
)
