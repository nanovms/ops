package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

func downloadLocalPackage(pkg string, config *types.Config) string {
	packagesDirPath := localPackageDirectoryPath()
	return downloadAndExtractPackage(packagesDirPath, pkg, config)
}

func localPackageDirectoryPath() string {
	return path.Join(api.GetOpsHome(), "local_packages")
}

func packageDirectoryPath() string {
	return path.Join(api.GetOpsHome(), "packages")
}

func downloadPackage(pkg string, config *types.Config) string {
	return downloadAndExtractPackage(packageDirectoryPath(), pkg, config)
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

func extractFilePackage(pkg string, name string, config *types.Config) string {
	f, err := os.Stat(pkg)
	if err != nil {
		log.Fatal(err)
	}

	if !f.IsDir() {
		if strings.HasSuffix(pkg, ".tar.gz") {
			return extractArchivedPackage(pkg, path.Join(localPackageDirectoryPath(), name), config)
		}

		log.Fatalf("Unsupported file format. Supported formats: .tar.gz")
	}

	tempDirectory, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}

	copyDirectory(pkg, tempDirectory)
	return movePackageFiles(tempDirectory, path.Join(localPackageDirectoryPath(), name))
}

func extractArchivedPackage(pkg string, target string, config *types.Config) string {
	tempDirectory, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(pkg, tempDirectory, config)
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

func downloadAndExtractPackage(packagesDirPath, pkg string, config *types.Config) string {
	err := os.MkdirAll(packagesDirPath, 0755)
	if err != nil {
		log.Fatal(err)
	}

	expackage := path.Join(packagesDirPath, pkg)
	opsPackage, err := api.DownloadPackage(pkg, config)
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(opsPackage, packagesDirPath, config)

	err = os.Remove(opsPackage)
	if err != nil {
		log.Fatal(err)
	}

	return expackage
}
