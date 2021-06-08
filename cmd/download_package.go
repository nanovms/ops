package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/mholt/archiver/v3"
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

func downloadLocalPackage(pkg string) string {
	packagesDirPath := localPackageDirectoryPath()
	return downloadAndExtractPackage(packagesDirPath, pkg)
}

func localPackageDirectoryPath() string {
	return path.Join(api.GetOpsHome(), "local_packages")
}

func endsWith(str string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(str, suffix) {
			return true
		}
	}

	return false
}

func startsWith(str string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}

	return false
}

func packageDirectoryPath() string {
	return path.Join(api.GetOpsHome(), "packages")
}

func downloadPackage(pkg string) string {
	return downloadAndExtractPackage(packageDirectoryPath(), pkg)
}

func extractFilePackage(pkg string, name string) string {
	f, err := os.Stat(pkg)

	if err == nil {
		if !f.IsDir() {
			supportedFormats := []string{".zip", ".tar", ".tar.gz", ".rar"}

			if endsWith(pkg, supportedFormats) {
				return extractZippedPackage(pkg, path.Join(localPackageDirectoryPath(), name))
			} else {
				log.Fatalf("Unsupported file format. Supported formats: ", strings.Join(supportedFormats, ", "))
			}
		} else {
			return movePackageFiles(pkg, path.Join(localPackageDirectoryPath(), name))
		}
	}

	log.Fatal(err)
	return ""
}

func extractZippedPackage(pkg string, target string) string {
	tempDirectory, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}

	err = archiver.Unarchive(pkg, tempDirectory)
	if err != nil {
		log.Fatal(err)
	}

	return movePackageFiles(tempDirectory, target)
}

func movePackageFiles(origin string, target string) string {
	manifestPath := path.Join(origin, "package.manifest")
	pkgConfig := &types.Config{}

	err := unWarpConfig(manifestPath, pkgConfig)
	if err != nil {
		log.Fatal(err)
	}

	os.RemoveAll(target)
	err = os.MkdirAll(target, 0755)
	if err != nil {
		log.Fatal(err)
	}

	files, err := ioutil.ReadDir(origin)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		os.Rename(path.Join(origin, f.Name()), path.Join(target, f.Name()))
	}

	return target
}

func downloadAndExtractPackage(packagesDirPath, pkg string) string {
	err := os.MkdirAll(packagesDirPath, 0755)
	if err != nil {
		log.Fatal(err)
	}

	expackage := path.Join(packagesDirPath, pkg)
	opsPackage, err := api.DownloadPackage(pkg)
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(opsPackage, packagesDirPath)

	err = os.Remove(opsPackage)
	if err != nil {
		log.Fatal(err)
	}

	return expackage
}
