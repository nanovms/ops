package lepton

import (
	"archive/tar"
	"crypto/sha256"

	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/nanovms/ops/fs"
)

// PackageList contains a list of known packages.
type PackageList struct {
	list map[string]Package
}

// Package is the definition of an OPS package.
type Package struct {
	Runtime     string `json:"runtime"`
	Version     string `json:"version"`
	Language    string `json:"language"`
	Description string `json:"description,omitempty"`
	SHA256      string `json:"sha256"`
}

// DownloadPackage downloads package by name
func DownloadPackage(name string) (string, error) {
	packages, err := GetPackageList()
	if err != nil {
		return "", nil
	}

	if _, ok := (*packages)[name]; !ok {
		return "", fmt.Errorf("package %q does not exist", name)
	}

	archivename := name + ".tar.gz"
	packagepath := path.Join(PackagesCache, archivename)
	if _, err := os.Stat(packagepath); os.IsNotExist(err) {
		if err = DownloadFileWithProgress(packagepath,
			fmt.Sprintf(PackageBaseURL, archivename), 600); err != nil {
			return "", err
		}
	}
	return packagepath, nil
}

// GetPackageList provides list of packages
func GetPackageList() (*map[string]Package, error) {
	var err error

	packageManifest := GetPackageManifestFile()
	stat, err := os.Stat(packageManifest)
	if os.IsNotExist(err) || PackageManifestChanged(stat, PackageManifestURL) {
		if err = DownloadFile(packageManifest, PackageManifestURL, 10, false); err != nil {
			return nil, err
		}
	}

	var packages PackageList
	data, err := ioutil.ReadFile(packageManifest)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &packages.list)
	if err != nil {
		return nil, err
	}

	return &packages.list, nil
}

// GetLocalPackageList provides list of local packages
func GetLocalPackageList() (*map[string]Package, error) {
	packages := map[string]Package{}

	localPackagesDir := GetOpsHome() + "/local_packages"

	localPackages, err := ioutil.ReadDir(localPackagesDir)
	if err != nil {
		return nil, err
	}

	for _, pkg := range localPackages {
		pkgName := pkg.Name()

		// ignore packages compressed
		if !strings.Contains(pkgName, "tar.gz") {
			data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/package.manifest", localPackagesDir, pkgName))
			if err != nil {
				return nil, err
			}

			var pkg Package
			err = json.Unmarshal(data, &pkg)
			if err != nil {
				return nil, err
			}

			packages[pkgName] = pkg
		}
	}

	return &packages, nil
}

func getPackageCache() string {
	packagefolder := path.Join(GetOpsHome(), "packages")
	if _, err := os.Stat(packagefolder); os.IsNotExist(err) {
		os.MkdirAll(packagefolder, 0755)
	}
	return packagefolder
}

// GetPackageManifestFile give path for package manifest file
func GetPackageManifestFile() string {
	return path.Join(getPackageCache(), PackageManifestFileName)
}

// PackageManifestChanged verifies if package manifest changed
func PackageManifestChanged(fino os.FileInfo, remoteURL string) bool {
	res, err := http.Head(remoteURL)
	if err != nil {
		if err, ok := err.(net.Error); ok {
			fmt.Printf(WarningColor, "missing internet?, using local manifest.\n")
		} else {
			panic(err)
		}

		return false
	}

	return fino.Size() != res.ContentLength
}

func sha256Of(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// ExtractPackage extracts package in ops home.
// This function is currently over-loaded.
func ExtractPackage(archive string, dest string) {
	sha := sha256Of(archive)

	// hack
	// this only verifies for packages - unfortunately this function is
	// used for extracting releases (which currently don't have
	// checksums)
	if strings.Contains(archive, ".ops/packages") {

		fname := filepath.Base(archive)
		fname = strings.ReplaceAll(fname, ".tar.gz", "")

		list, err := GetPackageList()
		if err != nil {
			panic(err)
		}

		if (*list)[fname].SHA256 != sha {
			fmt.Println("This package doesn't match what is in the manifest.")
			os.Exit(1)
		}

	}

	in, err := os.Open(archive)
	if err != nil {
		panic(err)
	}
	gzip, err := gzip.NewReader(in)
	if err != nil {
		panic(err)
	}
	defer gzip.Close()
	tr := tar.NewReader(gzip)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		if header == nil {
			continue
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					panic(err)
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				panic(err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				panic(err)
			}
			f.Close()
		}
	}
}

// BuildImageFromPackage builds nanos image using a package
func BuildImageFromPackage(packagepath string, c Config) error {
	m, err := BuildPackageManifest(packagepath, &c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if c.RunConfig.IPAddr != "" {
		m.AddNetworkConfig(&fs.ManifestNetworkConfig{
			IP:      c.RunConfig.IPAddr,
			Gateway: c.RunConfig.Gateway,
			NetMask: c.RunConfig.NetMask,
		})
	}

	if err := buildImage(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}
