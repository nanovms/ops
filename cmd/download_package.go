package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
)

func downloadLocalPackage(pkg string, config *types.Config) (string, error) {
	packagesDirPath := api.LocalPackagesRoot
	return downloadAndExtractPackage(packagesDirPath, pkg, config)
}

func downloadPackage(pkg string, config *types.Config) (string, error) {
	return downloadAndExtractPackage(api.PackagesRoot, pkg, config)
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
	var fds []os.DirEntry
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = os.ReadDir(src); err != nil {
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

func extractFilePackage(pkg string, name string, parch string, config *types.Config) string {
	f, err := os.Stat(pkg)
	if err != nil {
		log.Fatal(err)
	}

	ppath := path.Join(api.LocalPackagesRoot, parch, name)

	if !f.IsDir() {
		if strings.HasSuffix(pkg, ".tar.gz") {
			return extractArchivedPackage(pkg, ppath, config)
		}

		log.Fatalf("Unsupported file format. Supported formats: .tar.gz")
	}

	tempDirectory, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}

	copyDirectory(pkg, tempDirectory)
	return MovePackageFiles(tempDirectory, ppath)
}

func extractArchivedPackage(pkg string, target string, config *types.Config) string {
	tempDirectory, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(pkg, tempDirectory, config)
	return MovePackageFiles(tempDirectory, target)
}

// MovePackageFiles moves a package from a directory to another
func MovePackageFiles(origin string, target string) string {
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

	files, err := os.ReadDir(origin)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		err = xrename(path.Join(origin, f.Name()), path.Join(target, f.Name()))
		if err != nil {
			fmt.Println(err)
		}
	}

	return target
}

func xrename(srcPath, destPath string) error {
	err := os.Rename(srcPath, destPath)
	if err == nil {
		return nil
	}

	if linkErr, ok := err.(*os.LinkError); ok && linkErr.Op == "rename" {
		fi, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			dirEntries, err := os.ReadDir(srcPath)
			if err != nil {
				fmt.Println(err)
			}
			err = os.Mkdir(destPath, 0755)
			if err != nil {
				fmt.Println(err)
			}

			for _, entry := range dirEntries {
				err = xrename(srcPath+"/"+entry.Name(), destPath+"/"+entry.Name())
				if err != nil {
					fmt.Println(err)
				}
			}

		} else {

			src, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				os.Remove(destPath)
				return err
			}

			err = os.Remove(srcPath)
			if err != nil {
				return err
			}
			return nil
		}

		return nil
	}
	return err
}

func downloadAndExtractPackage(packagesDirPath, pkg string, config *types.Config) (string, error) {
	err := os.MkdirAll(packagesDirPath, 0755)
	if err != nil {
		return "", err
	}

	expackage := path.Join(packagesDirPath, strings.ReplaceAll(pkg, ":", "_"))
	opsPackage, err := api.DownloadPackage(pkg, config)
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(opsPackage, path.Dir(expackage), config)

	err = os.Remove(opsPackage)
	if err != nil {
		return "", err
	}

	return expackage, nil
}
