package lepton

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/go-errors/errors"
)

func DownloadPackage(name string) string {

	archivename := name + ".tar.gz"
	packagepath := path.Join(PackagesCache, archivename)
	if _, err := os.Stat(packagepath); os.IsNotExist(err) {
		if err = DownloadFile(packagepath,
			fmt.Sprintf(PackageBaseURL, archivename), 600); err != nil {
			panic(err)
		}
	}
	return packagepath
}

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
