package lepton

import (
	"archive/tar"
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

	"github.com/go-errors/errors"
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
	MD5         string `json:"md5,omitempty"`
}

// DownloadPackage downloads package by name
func DownloadPackage(name string) (string, error) {
	if _, ok := (*GetPackageList())[name]; !ok {
		return "", fmt.Errorf("package %q does not exist", name)
	}

	archivename := name + ".tar.gz"
	packagepath := path.Join(PackagesCache, archivename)
	if _, err := os.Stat(packagepath); os.IsNotExist(err) {
		if err = DownloadFile(packagepath,
			fmt.Sprintf(PackageBaseURL, archivename), 600); err != nil {
			return "", err
		}
	}
	return packagepath, nil
}

// GetPackageList provides list of packages
func GetPackageList() *map[string]Package {
	var err error

	packageManifest := GetPackageManifestFile()
	stat, err := os.Stat(packageManifest)
	if os.IsNotExist(err) || PackageManifestChanged(stat, PackageManifestURL) {
		if err = DownloadFile(packageManifest, PackageManifestURL, 10); err != nil {
			panic(err)
		}
	}

	var packages PackageList
	data, err := ioutil.ReadFile(packageManifest)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &packages.list)
	if err != nil {
		fmt.Println(err)
	}

	return &packages.list
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

// ExtractPackage extracts package in ops home
func ExtractPackage(archive string, dest string) {
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
	if err := buildImage(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}
