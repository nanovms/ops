package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// Unzip extracts given zipFile to destPath.
// destPath should be path to a directory.
func Unzip(zipFile, destPath string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if err := extractZippedFile(file, destPath); err != nil {
			return err
		}
	}
	return nil
}

// Extracts given zipped file to destPath.
// destPath should be path to a directory.
func extractZippedFile(file *zip.File, destPath string) error {
	targetPath := filepath.Join(destPath, file.Name)
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
			return err
		}
		return nil
	}

	baseDir := filepath.Dir(targetPath)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if err = os.MkdirAll(baseDir, 0755); err != nil {
			return err
		}
	}

	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	zFile, err := file.Open()
	if err != nil {
		return err
	}
	defer zFile.Close()

	if _, err := io.Copy(targetFile, zFile); err != nil {
		return err
	}
	return nil
}
