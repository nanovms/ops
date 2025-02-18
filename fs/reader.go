package fs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Reader allows reading filesystem contents from an image
type Reader struct {
	imageFile *os.File
	rootFS    *tfs
}

// Stat retrieves information for a file in the image
func (r *Reader) Stat(path string) (os.FileInfo, error) {
	return r.rootFS.stat(path)
}

// ReadDir returns the contents of a directory in the image
func (r *Reader) ReadDir(path string) ([]os.FileInfo, error) {
	return r.rootFS.readDir(path)
}

// ReadLink returns the target of a symbolic link in the image
func (r *Reader) ReadLink(path string) (string, error) {
	return r.rootFS.readLink(path)
}

// ReadFile returns the io.Reader of a file in the image.
func (r *Reader) ReadFile(path string) (io.Reader, error) {
	return r.rootFS.fileReader(path)
}

// CopyFile copies a file from the image to the local filesystem
func (r *Reader) CopyFile(src, dest string, dereference bool) error {
	if !dereference {
		fileInfo, err := r.rootFS.stat(src)
		if err != nil {
			return fmt.Errorf("cannot stat source file: %v", err)
		}
		if fileInfo.Mode() == os.ModeSymlink {
			target, err := r.rootFS.readLink(src)
			if err != nil {
				return fmt.Errorf("cannot read link target from source file: %v", err)
			}
			os.Remove(dest)
			err = os.Symlink(target, dest)
			if err != nil {
				return fmt.Errorf("cannot create symbolic link: %v", err)
			}
			return nil
		}
	}
	srcReader, err := r.rootFS.fileReader(src)
	if err != nil {
		return fmt.Errorf("cannot read source file: %v", err)
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("cannot create destination file: %v", err)
	}
	_, err = io.Copy(destFile, srcReader)
	destFile.Close()
	return err
}

// ListEnv returns the set of environment variables in the image
func (r *Reader) ListEnv() map[string]string {
	t := r.rootFS.getTuple("environment")
	envVars := make(map[string]string)
	if t != nil {
		for name, value := range *t {
			sValue, sOk := value.(string)
			if sOk {
				envVars[name] = sValue
			}
		}
	}
	return envVars
}

// GetUUID of image file system
func (r *Reader) GetUUID() string {
	return r.rootFS.getUUID()
}

// GetLabel of image file system
func (r *Reader) GetLabel() string {
	return r.rootFS.label
}

// Close closes the image file
func (r *Reader) Close() error {
	return r.imageFile.Close()
}

// NewReader returns an instance of Reader
func NewReader(imagePath string) (*Reader, error) {
	imageFile, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open image file: %v", err)
	}
	var info os.FileInfo
	info, err = imageFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot read image file: %v", err)
	}
	mbr := make([]byte, sectorSize)
	_, err = imageFile.Read(mbr)
	if err != nil {
		return nil, fmt.Errorf("cannot read MBR: %v", err)
	}
	var fsStart, fsSize uint64
	if (mbr[sectorSize-2] != 0x55) || (mbr[sectorSize-1] != 0xAA) { // assume raw filesystem
		fsStart = 0
		fsSize = uint64(info.Size())
	} else {
		part := getPartition(mbr, 0)
		partType := part[4]
		var rootFSPart int
		if partType == 0xEF { // EFI System Partition
			rootFSPart = 2 // 0 - uefi, 1 - bootfs, 2 - rootfs
		} else {
			rootFSPart = 1 // 0 - bootfs, 1 - rootfs
		}
		part = getPartition(mbr, rootFSPart)
		var lbaStart, sectors uint32
		binary.Read(bytes.NewReader(part[8:12]), binary.LittleEndian, &lbaStart)
		binary.Read(bytes.NewReader(part[12:16]), binary.LittleEndian, &sectors)
		if lbaStart == 0 || sectors == 0 { // assume raw filesystem
			fsStart = 0
			fsSize = uint64(info.Size())
		} else {
			fsStart = uint64(lbaStart) * sectorSize
			fsSize = uint64(sectors) * sectorSize
		}
	}
	reader := &Reader{
		imageFile: imageFile,
	}
	reader.rootFS, err = tfsRead(imageFile, fsStart, fsSize)
	return reader, err
}

// NewReaderBootFS returns an instance of Reader for bootFS
func NewReaderBootFS(imagePath string) (*Reader, error) {
	imageFile, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open image file: %v", err)
	}
	mbr := make([]byte, sectorSize)
	if _, err = imageFile.Read(mbr); err != nil {
		return nil, fmt.Errorf("cannot read MBR: %v", err)
	}
	if (mbr[sectorSize-2] != 0x55) || (mbr[sectorSize-1] != 0xAA) { // assume raw filesystem
		return nil, fmt.Errorf("%s", "bootfs not found")
	}
	part := getPartition(mbr, 0)
	partType := part[4]
	if partType == 0xEF { // EFI System Partition
		part = getPartition(mbr, 1) // 0 - uefi, 1 - bootfs, 2 - rootfs
	}
	var lbaStart, sectors uint32
	binary.Read(bytes.NewReader(part[8:12]), binary.LittleEndian, &lbaStart)
	binary.Read(bytes.NewReader(part[12:16]), binary.LittleEndian, &sectors)
	if lbaStart == 0 || sectors == 0 { // assume raw filesystem
		return nil, fmt.Errorf("%s", "bootfs not found")
	}
	bootFSStart := uint64(lbaStart) * sectorSize
	reader := &Reader{
		imageFile: imageFile,
	}
	reader.rootFS, err = tfsRead(imageFile, bootFSStart, bootFSSize)
	return reader, err
}

func getPartition(mbr []byte, index int) []byte {
	partStart := sectorSize - 2 - (4-index)*partitionEntrySize
	return mbr[partStart : partStart+partitionEntrySize]
}
