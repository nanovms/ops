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

// file system manifest
const manifest string = `(
    #64 bit elf to boot from host
    children:(kernel:(contents:(host:%v))
              #user program
              %v:(contents:(host:%v)))
    # filesystem path to elf for kernel to run
    program:/%v
    fault:t
    arguments:[%v sec third]
    environment:(USER:bobby PWD:/)
)`

const Version = "0.4"
const OpsReleaseUrl = "https://storage.googleapis.com/cli/%v/ops"

const ReleaseBaseUrl string = "https://storage.googleapis.com/nanos/release/"
const NightlyReleaseBaseUrl string = "https://storage.googleapis.com/nanos/release/nightly/"
const PackageBaseURL string = "https://storage.googleapis.com/packagehub/%v"
const PackageManifestURL string = "https://storage.googleapis.com/packagehub/manifest.json"
const PackageManifestFileName string = "manifest.json"
const mergedImg string = "tempimage"

const GCPStorageURL string = "https://storage.googleapis.com/%v/%v"

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

var PackagesCache string = getPackageCache()

func GetOpsHome() string {
	home, err := HomeDir()
	if err != nil {
		panic(err)
	}

	opshome := path.Join(home, ".ops")
	images := path.Join(opshome, ".ops", "images")
	if _, err := os.Stat(images); os.IsNotExist(err) {
		os.MkdirAll(images, 0755)
	}
	return opshome
}

func getImageTempDir(program string) string {
	temp := filepath.Base(program) + "_temp"
	path := path.Join(GetOpsHome(), temp)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
	return path
}

var NightlyReleaseUrl string = nightlyReleaseUrl()

func nightlyFileName() string {
	return fmt.Sprintf("nanos-nightly-%v.tar.gz", runtime.GOOS)
}

func nightlyReleaseUrl() string {
	var sb strings.Builder
	sb.WriteString(NightlyReleaseBaseUrl)
	sb.WriteString(nightlyFileName())
	return sb.String()
}

func nightlyLocalFolder() string {
	return path.Join(GetOpsHome(), "nightly")
}

var NightlyLocalFolder string = nightlyLocalFolder()

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

func RemoteTimeStamp() (string, error) {
	timestamp := fmt.Sprintf("nanos-nightly-%v.timestamp", runtime.GOOS)
	resp, err := http.Get(NightlyReleaseBaseUrl + timestamp)
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

var LatestReleaseVersion string = getLatestRelVersion()

const (
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
)

func getLatestRelVersion() string {
	resp, err := http.Get(ReleaseBaseUrl + "latest.txt")
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

var LocalReleaseVersion string = getLocalRelVersion()

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

func getReleaseUrl(version string) string {
	var sb strings.Builder
	sb.WriteString(ReleaseBaseUrl)
	sb.WriteString(version)
	sb.WriteRune('/')
	sb.WriteString(releaseFileName(version))
	return sb.String()
}

func getReleaseLocalFolder(version string) string {
	return path.Join(GetOpsHome(), version)
}

const (
	commonArchive = "https://storage.googleapis.com/nanos/common/common.tar.gz"
	libDNS        = "/lib/x86_64-linux-gnu/libnss_dns.so.2"
	sslCERT       = "/etc/ssl/certs/ca-certificates.crt"
)
