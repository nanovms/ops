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
	"github.com/nanovms/ops/constants"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
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
func DownloadPackage(name string, config *types.Config) (string, error) {
	packages, err := GetPackageList(config)
	if err != nil {
		return "", nil
	}

	if _, ok := (*packages)[name]; !ok {
		return "", fmt.Errorf("package %q does not exist", name)
	}

	archivename := name + ".tar.gz"
	packagepath := path.Join(PackagesCache, archivename)
	_, err = os.Stat(packagepath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		return packagepath, nil
	}

	pkgBaseURL := PackageBaseURL

	// Check config override
	if config != nil {
		cPkgBaseURL := strings.Trim(config.PackageBaseURL, " ")
		if len(cPkgBaseURL) > 0 {
			pkgBaseURL = cPkgBaseURL
		}
	}

	// Check environment variable override
	ePkgBaseURL := os.Getenv("OPS_PACKAGE_BASE_URL")
	if len(ePkgBaseURL) > 0 {
		pkgBaseURL = ePkgBaseURL
	}

	isNetworkRepo := !strings.HasPrefix(pkgBaseURL, "file://")
	if isNetworkRepo {
		var fileURL string
		if strings.HasSuffix(pkgBaseURL, "/") {
			fileURL = pkgBaseURL + archivename
		} else {
			fileURL = fmt.Sprintf("%s/%s", pkgBaseURL, archivename)
		}

		if err = DownloadFileWithProgress(packagepath, fileURL, 600); err != nil {
			return "", err
		}

		return packagepath, nil
	}

	pkgBaseURL = strings.TrimPrefix(pkgBaseURL, "file://")
	srcPath := filepath.Join(pkgBaseURL, archivename)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return "", err
	}

	destFile, err := os.Create(packagepath)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	progressCounter := NewWriteCounter(int(srcStat.Size()))
	progressCounter.Start()
	_, err = io.Copy(destFile, io.TeeReader(srcFile, progressCounter))
	progressCounter.Finish()

	return packagepath, nil
}

// GetPackageList provides list of packages
func GetPackageList(config *types.Config) (*map[string]Package, error) {
	var err error

	pkgManifestURL := PackageManifestURL

	// Check config override
	if config != nil {
		cPkgManifestURL := strings.Trim(config.PackageManifestURL, " ")
		if len(cPkgManifestURL) > 0 {
			pkgManifestURL = cPkgManifestURL
		}
	}

	// Check environment var override
	ePkgManifestURL := os.Getenv("OPS_PACKAGE_MANIFEST_URL")
	if len(ePkgManifestURL) > 0 {
		pkgManifestURL = ePkgManifestURL
	}

	packageManifest := GetPackageManifestFile()
	if strings.HasPrefix(pkgManifestURL, "file://") {
		destFile, err := os.OpenFile(packageManifest, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
		defer destFile.Close()

		pkgManifestURL = strings.TrimPrefix(pkgManifestURL, "file://")
		srcFile, err := os.Open(pkgManifestURL)
		if err != nil {
			return nil, err
		}
		defer srcFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return nil, err
		}
	} else {
		stat, err := os.Stat(packageManifest)
		if os.IsNotExist(err) || PackageManifestChanged(stat, pkgManifestURL) {
			if err = DownloadFile(packageManifest, pkgManifestURL, 10, false); err != nil {
				return nil, err
			}
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
			fmt.Printf(constants.WarningColor, "missing internet?, using local manifest.\n")
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
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// ExtractPackage extracts package in ops home.
// This function is currently over-loaded.
func ExtractPackage(archive, dest string, config *types.Config) {
	sha := sha256Of(archive)

	// hack
	// this only verifies for packages - unfortunately this function is
	// used for extracting releases (which currently don't have
	// checksums)
	if strings.Contains(archive, ".ops/packages") {

		fname := filepath.Base(archive)
		fname = strings.ReplaceAll(fname, ".tar.gz", "")

		list, err := GetPackageList(config)
		if err != nil {
			panic(err)
		}

		if (*list)[fname].SHA256 != sha {
			log.Fatalf("This package doesn't match what is in the manifest.")
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
			if err := f.Truncate(0); err != nil {
				panic(err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				panic(err)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				log.Warn(err.Error())
			}
		}
	}
}

// BuildImageFromPackage builds nanos image using a package
func BuildImageFromPackage(packagepath string, c types.Config) error {
	m, err := BuildPackageManifest(packagepath, &c)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if err := createImageFile(&c, m); err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}
