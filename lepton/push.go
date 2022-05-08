package lepton

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// BuildRequestForArchiveUpload builds the request to upload a package with the provided metadata
func BuildRequestForArchiveUpload(namespace, name string, pkg Package, archiveLocation string, private bool) (*http.Request, error) {
	privateStr := "off"
	if private {
		privateStr = "on"
	}
	params := map[string]string{
		"name":        name,
		"description": pkg.Description,
		"language":    pkg.Language,
		"runtime":     pkg.Runtime,
		"version":     pkg.Version,
		"namespace":   namespace,
		"private":     privateStr,
	}
	return newfileUploadRequest(PkghubBaseURL+"/packages/create", params, "package", archiveLocation)

}

func newfileUploadRequest(uri string, params map[string]string, fileParamName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileParamName, fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := BaseHTTPRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", writer.Boundary()))
	return req, nil
}

// CreateTarGz builds a .tar.gz archive with the directory of the source as the root of the archive
func CreateTarGz(src string, destination string) error {
	fd, err := os.Create(destination)
	if err != nil {
		return err
	}
	// tar > gzip > buf
	zr := gzip.NewWriter(fd)
	tw := tar.NewWriter(zr)

	// is file a folder?
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		// get header
		header, err := tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// get content
		data, err := os.Open(src)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			return err
		}
	} else if mode.IsDir() { // folder

		// walk through every file in the folder
		filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			// generate tar header
			header, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			filename, _ := filepath.Rel(filepath.Dir(src), filepath.ToSlash(file))
			header.Name = filename

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, data); err != nil {
					return err
				}
			}
			return nil
		})
	} else {
		return fmt.Errorf("error: file type not supported")
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}

	return nil
}
