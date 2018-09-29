package lepton

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// BuildImage builds a unikernel image for user
// supplied ELF binary.
func BuildImage(userImage string, bootImage string) error {
	var err error
	if err = buildImage(userImage, bootImage); err != nil {
		return err
	}
	return nil
}

func createFile(filepath string) (*os.File, error) {
	path := path.Dir(filepath)
	var _, err = os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
	fd, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func buildImage(userImage string, finaImage string) error {
	//  prepare manifest file
	var elfname = filepath.Base(userImage)
	var extension = filepath.Ext(elfname)
	elfname = elfname[0 : len(elfname)-len(extension)]
	elfmanifest := fmt.Sprintf(manifest, kernelImg, elfname, userImage, elfname, elfname)

	// invoke mkfs to create the filesystem ie kernel + elf image
	mkfs := exec.Command("./mkfs", mergedImg)
	stdin, err := mkfs.StdinPipe()
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdin, elfmanifest)
	if err != nil {
		return err
	}
	out, err := mkfs.CombinedOutput()
	if err != nil {
		log.Println(out)
		return err
	}

	// produce final image, boot + kernel + elf
	fd, err := createFile(finaImage)
	defer fd.Close()
	if err != nil {
		return err
	}
	catcmd := exec.Command("cat", bootImg, mergedImg)
	catcmd.Stdout = fd
	err = catcmd.Start()
	if err != nil {
		return err
	}
	catcmd.Wait()
	return nil
}

type dummy struct {
	total uint64
}

func (bc dummy) Write(p []byte) (int, error) {
	return len(p), nil
}

func DownloadBootImages() error {
	return DownloadImages(dummy{})
}

// DownloadImages downloads latest kernel images.
func DownloadImages(w io.Writer) error {
	var err error
	if _, err := os.Stat("staging"); os.IsNotExist(err) {
		os.MkdirAll("staging", 0755)
	}

	if _, err = os.Stat("./mkfs"); os.IsNotExist(err) {
		if err = downloadFile("mkfs", fmt.Sprintf(bucketBaseUrl, "mkfs"), w); err != nil {
			return err
		}
	}

	// make mkfs executable
	err = os.Chmod("mkfs", 0775)
	if err != nil {
		return err
	}

	if _, err = os.Stat("staging/boot"); os.IsNotExist(err) {
		if err = downloadFile("staging/boot", fmt.Sprintf(bucketBaseUrl, "boot"), w); err != nil {
			return err
		}
	}

	if _, err = os.Stat("staging/stage3"); os.IsNotExist(err) {
		if err = downloadFile("staging/stage3", fmt.Sprintf(bucketBaseUrl, "stage3"), w); err != nil {
			return err
		}
	}
	return nil
}

func downloadFile(filepath string, url string, w io.Writer) error {
	// download to a temp file and later rename it
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// progress reporter.
	_, err = io.Copy(out, io.TeeReader(resp.Body, w))
	if err != nil {
		return err
	}
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}
	return nil
}
