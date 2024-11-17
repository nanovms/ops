package lepton

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nanovms/ops/constants"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

// Version for ops
var Version string

// OpsReleaseURL gives URL to download latest ops binary
const OpsReleaseURL = "https://storage.googleapis.com/cli/%v/ops"

const releaseBaseURL string = "https://storage.googleapis.com/nanos/release/"

const nightlyReleaseBaseURL string = "https://storage.googleapis.com/nanos/release/nightly/"

// PackageBaseURL gives URL for downloading of packages
const PackageBaseURL string = PkghubBaseURL + "/v2/packages"

// PackageManifestURL stores info about all packages
const PackageManifestURL string = PkghubBaseURL + "/v2/manifest.json"

// PackageManifestFileName is manifest file path
const PackageManifestFileName string = "manifest.json"

// PkghubBaseURL is the base url of packagehub
const PkghubBaseURL string = "https://repo.ops.city"

var (
	// LocalVolumeDir is the default local volume directory
	LocalVolumeDir = path.Join(GetOpsHome(), "volumes")
)

// GenerateImageName generate image name
func GenerateImageName(program string) string {
	program = filepath.Base(program)
	images := path.Join(GetOpsHome(), "images")
	return fmt.Sprintf("%s/%s", images, program)
}

// GetOpsHome get ops directory path
// We store all ops related info, packages, images in this directory
//
// TODO: this needs to be memoized.
func GetOpsHome() string {
	home, err := HomeDir()
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
	opshome := path.Join(home, ".ops")

	// Check home directory override via OPS_HOME environment variable.
	// OPS_HOME should not point to .ops directory replacement,
	// but should point to the parent directory in which .ops directory located,
	// so the homedir path after OPS_HOME set should be OPS_HOME/.ops.
	envOpsHome := os.Getenv("OPS_HOME")
	if envOpsHome != "" {
		altHomeDir := filepath.Join(envOpsHome, ".ops")
		if _, err := os.Stat(altHomeDir); os.IsNotExist(err) {
			if err = os.MkdirAll(altHomeDir, 0755); err != nil {
				fmt.Println("failed to create OPS home directory at ", altHomeDir)
				exitWithError(err.Error())
			}
		}
		opshome = altHomeDir
	}

	// it prob. makes sense to join the networks/composes together and
	// have compose be optional on network.

	images := path.Join(opshome, "images")
	instances := path.Join(opshome, "instances")
	networks := path.Join(opshome, "networks")
	composes := path.Join(opshome, "composes")
	manifests := path.Join(opshome, "manifests")
	volumes := path.Join(opshome, "volumes")

	// memoize me
	localPackages := path.Join(opshome, "local_packages")
	packages := path.Join(opshome, "packages")

	if _, err := os.Stat(images); os.IsNotExist(err) {
		os.MkdirAll(images, 0755)
	}

	if _, err := os.Stat(instances); os.IsNotExist(err) {
		os.MkdirAll(instances, 0755)
	}

	if _, err := os.Stat(manifests); os.IsNotExist(err) {
		os.MkdirAll(manifests, 0755)
	}

	if _, err := os.Stat(networks); os.IsNotExist(err) {
		os.MkdirAll(networks, 0755)
	}

	if _, err := os.Stat(composes); os.IsNotExist(err) {
		os.MkdirAll(composes, 0755)
	}

	if _, err := os.Stat(volumes); os.IsNotExist(err) {
		os.MkdirAll(volumes, 0755)
	}

	if _, err := os.Stat(localPackages); os.IsNotExist(err) {
		os.MkdirAll(localPackages, 0755)
	}

	if _, err := os.Stat(packages); os.IsNotExist(err) {
		os.MkdirAll(packages, 0755)
	}

	return opshome
}

func getImageTempDir(c *types.Config) string {
	temp := filepath.Base(c.Program) + "_temp"

	if c.BuildDir == "" {
		dir, err := os.MkdirTemp("", temp)
		if err != nil {
			log.Error(err)
		}

		c.BuildDir = dir

	}

	return c.BuildDir
}

var realGOOS = getGOOS()

// RealGOARCH is the underlying host architecture.
var RealGOARCH = getGOARCH()

// AltGOARCH is an optional user-supplied cross-build arch for both build and run.
var AltGOARCH = ""

func getGOARCH() string {
	return runtime.GOARCH
}

func getGOOS() string {
	goos := runtime.GOOS
	if goos == "freebsd" {
		return "linux"
	}

	return goos
}

// LocalTimeStamp gives local timestamp from download nightly build
func LocalTimeStamp() (string, error) {
	timestamp := nightlyTimestamp()
	data, err := os.ReadFile(path.Join(NightlyLocalFolderm, timestamp))

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
	timestamp := nightlyTimestamp()
	resp, err := http.Get(nightlyReleaseBaseURL + timestamp)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func updateLocalTimestamp(timestamp string) error {
	fname := nightlyTimestamp()
	return os.WriteFile(path.Join(NightlyLocalFolderm, fname), []byte(timestamp), 0755)
}

// UpdateLocalRelease updates nanos version used on ops operations
func UpdateLocalRelease(version string) error {
	local := path.Join(GetOpsHome(), "latest.txt")
	LocalReleaseVersion = version
	return os.WriteFile(local, []byte(version), 0755)
}

// LatestReleaseVersion give latest stable release for nanos
var LatestReleaseVersion = getLatestRelVersion()

func getLatestRelVersion() string {
	resp, err := http.Get(releaseBaseURL + "latest.txt")
	if err != nil {
		fmt.Printf(constants.WarningColor, "version lookup failed, using local.\n")
		if LocalReleaseVersion == "0.0" {
			log.Fatalf(constants.ErrorColor, "No local build found.")
		}
		return LocalReleaseVersion
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return strings.TrimSuffix(string(data), "\n")
}

// LocalReleaseVersion is version latest release downloaded in ops home
var LocalReleaseVersion = getLocalRelVersion()

func getLocalRelVersion() string {
	data, err := os.ReadFile(path.Join(GetOpsHome(), "latest.txt"))
	// nothing yet, force a download
	if os.IsNotExist(err) {
		return "0.0"
	}
	return strings.TrimSuffix(string(data), "\n")
}

func releaseFileName(version string) string {
	return fmt.Sprintf("nanos-release-%v-%v.tar.gz", realGOOS, version)
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

func getKlibsDir(nightly bool, nanosVersion string) string {
	if nightly {
		return NightlyLocalFolder() + "/klibs"
	} else if nanosVersion != "" && nanosVersion != "0.0" {
		return getReleaseLocalFolder(nanosVersion) + "/klibs"
	}

	return getLastReleaseLocalFolder() + "/klibs"
}

// GetUefiBoot retrieves UEFI bootloader file path, if found
func GetUefiBoot(version string) string {
	folder := getReleaseLocalFolder(version)
	f, err := os.Open(folder)
	if err != nil {
		return ""
	}
	defer f.Close()
	var files []os.FileInfo
	files, err = f.Readdir(0)
	if err != nil {
		return ""
	}
	for _, file := range files {
		if file.Mode().IsRegular() && strings.HasSuffix(file.Name(), ".efi") {
			return path.Join(folder, file.Name())
		}
	}
	return ""
}

const (
	commonArchive = "https://storage.googleapis.com/nanos/common/common.tar.gz"
	libDNS        = "/lib/x86_64-linux-gnu/libnss_dns.so.2"
	sslCERT       = "/etc/ssl/certs/ca-certificates.crt"
)
