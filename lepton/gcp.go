package lepton

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type GCloud struct{}

func (p *GCloud) BuildImage(ctx *Context) error {
	c := ctx.config
	err := BuildImage(*c)
	if err != nil {
		return err
	}

	imagePath := c.RunConfig.Imagename
	symlink := filepath.Join(filepath.Dir(imagePath), "disk.raw")

	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return fmt.Errorf("failed to unlink: %+v", err)
		}
	} else if os.IsNotExist(err) {
		return fmt.Errorf("failed to check symlink: %+v", err)
	}

	err = os.Symlink(imagePath, symlink)
	if err != nil {
		return err
	}
	// name the gcp archive
	name := fmt.Sprintf("nanos-%v-image.tar.gz", filepath.Base(c.Program))
	archPath := filepath.Join(filepath.Dir(imagePath), name)
	files := []string{symlink}

	err = createArchive(archPath, files)
	if err != nil {
		return err
	}
	return nil
}

func (p *GCloud) DeployImage(ctx *Context) error {
	return nil
}
func (p *GCloud) CreateInstance(ctx *Context) error {
	return nil
}

func createArchive(archive string, files []string) error {
	fd, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer fd.Close()
	gzw := gzip.NewWriter(fd)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(stat, stat.Name())
		if err != nil {
			return err
		}
		// update the name to correctly
		header.Name = filepath.Base(file)

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		fi, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data to tar
		if _, err := io.Copy(tw, fi); err != nil {
			return err
		}
		fi.Close()
	}
	return nil
}
