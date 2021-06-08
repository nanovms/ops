package cmd

import (
	"io"
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

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func copyDirectory(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = copyDirectory(srcfp, dstfp); err != nil {
				log.Fatal(err)
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}

func extractFilePackage(pkg string, name string) string {
	f, err := os.Stat(pkg)

	if err == nil {
		if !f.IsDir() {
			supportedFormats := []string{".zip", ".tar", ".tar.gz", ".rar"}

			if endsWith(pkg, supportedFormats) {
				return extractZippedPackage(pkg, path.Join(localPackageDirectoryPath(), name))
			}

			log.Fatalf("Unsupported file format. Supported formats: ", strings.Join(supportedFormats, ", "))
		}

		tempDirectory, err := ioutil.TempDir("", "*")
		if err != nil {
			log.Fatal(err)
		}

		copyDirectory(pkg, tempDirectory)
		return movePackageFiles(tempDirectory, path.Join(localPackageDirectoryPath(), name))
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
