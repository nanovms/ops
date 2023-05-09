package fs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-errors/errors"
)

const sectorSize = 512
const sectorsPerTrack = 63
const heads = 255
const maxCyl = 1023

const regionFilesystem = 12

const partitionEntrySize = 16

const klogDumpSize = 4 * 1024
const uefiFSSize = 33 * 1024 * 1024
const bootFSSize = 12 * 1024 * 1024

/* Volume Boot Record */
var uefiVBR = []byte{
	0xEB, 0x58, 0x90, 0x6D, 0x6B, 0x66, 0x73, 0x2E, 0x66, 0x61, 0x74, 0x00, 0x02, 0x01, 0x20, 0x00,
	0x02, 0x00, 0x00, 0x00, 0x00, 0xF8, 0x00, 0x00, 0x20, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xCC, 0x06, 0x01, 0x00, 0x06, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x80, 0x00, 0x29, 0x9E, 0xF9, 0x43, 0xDF, 0x4E, 0x4F, 0x20, 0x4E, 0x41, 0x4D, 0x45, 0x20, 0x20,
	0x20, 0x20, 0x46, 0x41, 0x54, 0x33, 0x32, 0x20, 0x20, 0x20, 0x0E, 0x1F, 0xBE, 0x77, 0x7C, 0xAC,
	0x22, 0xC0, 0x74, 0x0B, 0x56, 0xB4, 0x0E, 0xBB, 0x07, 0x00, 0xCD, 0x10, 0x5E, 0xEB, 0xF0, 0x32,
	0xE4, 0xCD, 0x16, 0xCD, 0x19, 0xEB, 0xFE, 0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x6E,
	0x6F, 0x74, 0x20, 0x61, 0x20, 0x62, 0x6F, 0x6F, 0x74, 0x61, 0x62, 0x6C, 0x65, 0x20, 0x64, 0x69,
	0x73, 0x6B, 0x2E, 0x20, 0x20, 0x50, 0x6C, 0x65, 0x61, 0x73, 0x65, 0x20, 0x69, 0x6E, 0x73, 0x65,
	0x72, 0x74, 0x20, 0x61, 0x20, 0x62, 0x6F, 0x6F, 0x74, 0x61, 0x62, 0x6C, 0x65, 0x20, 0x66, 0x6C,
	0x6F, 0x70, 0x70, 0x79, 0x20, 0x61, 0x6E, 0x64, 0x0D, 0x0A, 0x70, 0x72, 0x65, 0x73, 0x73, 0x20,
	0x61, 0x6E, 0x79, 0x20, 0x6B, 0x65, 0x79, 0x20, 0x74, 0x6F, 0x20, 0x74, 0x72, 0x79, 0x20, 0x61,
	0x67, 0x61, 0x69, 0x6E, 0x20, 0x2E, 0x2E, 0x2E, 0x20, 0x0D, 0x0A,
}

/* FSInfo Structure */
var uefiFSInfo = []byte{
	0x52, 0x52, 0x61, 0x41, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x72, 0x72, 0x41, 0x61, 0xAE, 0x01, 0x01, 0x00, 0xF3,
}

/* First entries of File Allocation Table */
var uefiFAT = []byte{
	0xF8, 0xFF, 0xFF, 0x0F, 0xFF, 0xFF, 0xFF, 0x0F, 0xF8, 0xFF, 0xFF, 0x0F, 0xFF, 0xFF, 0xFF, 0x0F,
	0xFF, 0xFF, 0xFF, 0x0F,
}

/* "EFI" entry in root directory */
var uefiDirEfi = []byte{
	0x45, 0x46, 0x49, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x60, 0x39,
	0x7C, 0x52, 0x7C, 0x52, 0x00, 0x00, 0x60, 0x39, 0x7C, 0x52, 0x03,
}

/* "EFI" directory entries with "Boot" subfolder */
var uefiDirBoot = []byte{
	0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x2E, 0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x41, 0x42, 0x00, 0x6F, 0x00, 0x6F, 0x00, 0x74, 0x00, 0x00, 0x00, 0x0F, 0x00, 0xDD, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF,
	0x42, 0x4F, 0x4F, 0x54, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x04,
}

/* "EFI/Boot" directory entries with "bootx64.efi" file */
var uefiFileBootx64 = []byte{
	0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x2E, 0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x41, 0x62, 0x00, 0x6F, 0x00, 0x6F, 0x00, 0x74, 0x00, 0x78, 0x00, 0x0F, 0x00, 0x1D, 0x36, 0x00,
	0x34, 0x00, 0x2E, 0x00, 0x65, 0x00, 0x66, 0x00, 0x69, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF,
	0x42, 0x4F, 0x4F, 0x54, 0x58, 0x36, 0x34, 0x20, 0x45, 0x46, 0x49, 0x20, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x05, 0x00,
}

/* "EFI/Boot" directory entries with "bootaa64.efi" file */
var uefiFileBootaa64 = []byte{
	0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x2E, 0x2E, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x10, 0x00, 0x00, 0x8F, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x8F, 0x9C, 0x7B, 0x52, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x41, 0x62, 0x00, 0x6F, 0x00, 0x6F, 0x00, 0x74, 0x00, 0x61, 0x00, 0x0F, 0x00, 0x54, 0x61, 0x00,
	0x36, 0x00, 0x34, 0x00, 0x2E, 0x00, 0x65, 0x00, 0x66, 0x00, 0x00, 0x00, 0x69, 0x00, 0x00, 0x00,
	0x42, 0x4F, 0x4F, 0x54, 0x41, 0x41, 0x36, 0x34, 0x45, 0x46, 0x49, 0x20, 0x00, 0x64, 0x4B, 0x9C,
	0x7B, 0x52, 0x7B, 0x52, 0x00, 0x00, 0x4B, 0x9C, 0x7B, 0x52, 0x05, 0x00,
}

// MkfsCommand wraps mkfs calls
type MkfsCommand struct {
	bootPath   string
	uefiPath   string
	label      string
	manifest   *Manifest
	partitions bool
	size       int64
	outPath    string
	rootTfs    *tfs
}

// NewMkfsCommand returns an instance of MkfsCommand
func NewMkfsCommand(m *Manifest, partitions bool) *MkfsCommand {
	return &MkfsCommand{
		bootPath:   "",
		uefiPath:   "",
		label:      "",
		manifest:   m,
		partitions: partitions,
		size:       0,
		outPath:    "",
	}
}

// SetFileSystemSize adds argument that sets file system size
func (m *MkfsCommand) SetFileSystemSize(size string) error {
	f := func(c rune) bool {
		return !unicode.IsNumber(c)
	}
	unitsIndex := strings.IndexFunc(size, f)
	var mul int64
	if unitsIndex < 0 {
		mul = 1
		unitsIndex = len(size)
	} else if unitsIndex == 0 {
		return errors.New("invalid size " + size)
	} else {
		units := strings.ToLower(size[unitsIndex:])
		if units == "k" {
			mul = 1024
		} else if units == "m" {
			mul = 1024 * 1024
		} else if units == "g" {
			mul = 1024 * 1024 * 1024
		} else {
			return errors.New("invalid units " + units)
		}
	}
	var err error
	m.size, err = strconv.ParseInt(size[:unitsIndex], 10, 64)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	m.size *= mul
	sectors := (m.size + sectorSize - 1) / sectorSize
	m.size = sectors * sectorSize
	return nil
}

// SetBoot adds argument that sets file system boot
func (m *MkfsCommand) SetBoot(boot string) {
	m.bootPath = boot
	m.partitions = true
}

// SetUefi sets path of UEFI bootloader
func (m *MkfsCommand) SetUefi(uefi string) {
	m.uefiPath = uefi
	m.partitions = true
}

// SetFileSystemPath add argument that sets file system path
func (m *MkfsCommand) SetFileSystemPath(fsPath string) {
	m.outPath = fsPath
}

// SetLabel add label argument that sets file system label
func (m *MkfsCommand) SetLabel(label string) {
	m.label = label
}

// Execute runs mkfs command
func (m *MkfsCommand) Execute() error {
	if m.outPath == "" {
		return fmt.Errorf("output image file path not set")
	}
	var outFile *os.File
	var err error
	outFile, err = os.Create(m.outPath)
	if err != nil {
		return fmt.Errorf("cannot create output file %s: %v", m.outPath, err)
	}
	defer outFile.Close()
	var outOffset uint64
	var bootFile *os.File
	if m.bootPath != "" {
		bootFile, err = os.Open(m.bootPath)
		if err != nil {
			return fmt.Errorf("cannot open boot image %s: %v", m.bootPath, err)
		}
		defer bootFile.Close()
		b := make([]byte, 8192)
		for {
			n, err := bootFile.Read(b)
			if err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("cannot read boot image %s: %v", m.bootPath, err)
			}
			n, err = outFile.Write(b[:n])
			if err != nil {
				return fmt.Errorf("cannot write output file %s: %v", m.outPath, err)
			}
			outOffset += uint64(n)
		}
	} else if m.partitions {
		// Create a Master Boot Record with a partition table
		mbr := make([]byte, sectorSize)
		mbr[sectorSize-2] = 0x55
		mbr[sectorSize-1] = 0xAA
		_, err = outFile.Write(mbr)
		if err != nil {
			return fmt.Errorf("cannot write partition table: %v", err)
		}
		outOffset = sectorSize
	}
	if m.partitions {
		outOffset += klogDumpSize
	}
	if m.uefiPath != "" {
		outOffset, err = writeUefiPart(outFile, outOffset, m.uefiPath)
		if err != nil {
			return fmt.Errorf("cannot write UEFI partition: %v", err)
		}
	}
	manifest := m.manifest
	var root map[string]interface{}
	if manifest != nil {
		manifest.finalize()
		if manifest.boot != nil {
			_, err = tfsWrite(outFile, outOffset, bootFSSize, "", manifest.boot)
			if err != nil {
				return fmt.Errorf("cannot write boot filesystem: %v", err)
			}
			outOffset += bootFSSize
		}
		root = manifest.root
	} else {
		root = mkFS()
	}
	m.rootTfs, err = tfsWrite(outFile, outOffset, 0, m.label, root)
	if err != nil {
		return fmt.Errorf("cannot write root filesystem: %v", err)
	}
	if m.size != 0 {
		var info os.FileInfo
		info, err = outFile.Stat()
		if err != nil {
			return fmt.Errorf("cannot get size of output file: %v", err)
		}
		if info.Size() < m.size {
			err = outFile.Truncate(m.size)
			if err != nil {
				return fmt.Errorf("cannot set size of output file: %v", err)
			}
		}
	}
	if m.partitions {
		err = writeMBR(outFile, m.uefiPath != "")
		if err != nil {
			return fmt.Errorf("cannot write MBR: %v", err)
		}
	}
	return nil
}

// GetUUID returns the uuid of file system built
func (m *MkfsCommand) GetUUID() string {
	uuid := m.rootTfs.uuid

	/* UUID format: 00112233-4455-6677-8899-aabbccddeeff */
	var uuidStr string
	for i := 0; i < 4; i++ {
		uuidStr += fmt.Sprintf("%02x", uuid[i])
	}
	uuidStr += fmt.Sprintf("-%02x%02x-%02x%02x-%02x%02x-", uuid[4], uuid[5], uuid[6], uuid[7], uuid[8], uuid[9])
	for i := 10; i < 16; i++ {
		uuidStr += fmt.Sprintf("%02x", uuid[i])
	}
	return uuidStr
}

func mkFS() map[string]interface{} {
	root := make(map[string]interface{})
	root["children"] = make(map[string]interface{})
	return root
}

func getRootDir(root map[string]interface{}) map[string]interface{} {
	return root["children"].(map[string]interface{})
}

// Creates the EFI System Partition, i.e. a FAT32 filesystem with the UEFI loader file in the EFI/Boot directory.
func writeUefiPart(imgFile *os.File, offset uint64, uefiPath string) (uint64, error) {
	uefiLoader, err := os.Open(uefiPath)
	if err != nil {
		return offset, fmt.Errorf("cannot open UEFI loader file: %v", err)
	}
	defer uefiLoader.Close()
	_, err = imgFile.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return offset, err
	}
	err = writeBlobPadded(imgFile, uefiVBR, true)
	if err != nil {
		return offset, err
	}
	/* offset 0x200 */
	err = writeBlobPadded(imgFile, uefiFSInfo, true)
	if err != nil {
		return offset, err
	}
	/* offset 0x400 */
	_, err = imgFile.Seek(4*sectorSize, io.SeekCurrent)
	if err != nil {
		return offset, err
	}
	/* offset 0xC00 */
	err = writeBlobPadded(imgFile, uefiVBR, true)
	if err != nil {
		return offset, err
	}
	/* offset 0xE00 */
	_, err = imgFile.Seek(25*sectorSize, io.SeekCurrent)
	if err != nil {
		return offset, err
	}
	/* offset 0x4000 */
	_, err = imgFile.Write(uefiFAT)
	if err != nil {
		return offset, err
	}
	var fileInfo os.FileInfo
	fileInfo, err = uefiLoader.Stat()
	if err != nil {
		return offset, fmt.Errorf("failed to get UEFI loader file info: %v", err)
	}
	firstCluster := uint32(0x06)
	lastCluster := firstCluster + uint32((fileInfo.Size()+sectorSize-1)/sectorSize)
	var valArray [4]byte
	for cluster := firstCluster; cluster <= lastCluster; cluster++ {
		_ = binary.Write(bytes.NewBuffer(valArray[0:0]), binary.LittleEndian, cluster)
		_, err = imgFile.Write(valArray[:])
		if err != nil {
			return offset, err
		}
	}
	/* end of cluster chain */
	_ = binary.Write(bytes.NewBuffer(valArray[0:0]), binary.LittleEndian, uint32(0x0FFFFFFF))
	_, err = imgFile.Write(valArray[:])
	if err != nil {
		return offset, err
	}
	clusterChainSize := len(uefiFAT) + int(lastCluster-firstCluster+2)*len(valArray)
	_, err = imgFile.Seek(int64(0x81800-clusterChainSize), io.SeekCurrent)
	if err != nil {
		return offset, err
	}
	/* offset 0x85800 */
	err = writeBlobPadded(imgFile, uefiDirEfi, false)
	if err != nil {
		return offset, err
	}
	/* offset 0x85A00 */
	err = writeBlobPadded(imgFile, uefiDirBoot, false)
	if err != nil {
		return offset, err
	}
	/* offset 0x85C00 */
	fileName := filepath.Base(uefiPath)
	if fileName == "bootx64.efi" {
		_, err = imgFile.Write(uefiFileBootx64)
	} else if fileName == "bootaa64.efi" {
		_, err = imgFile.Write(uefiFileBootaa64)
	} else {
		return offset, fmt.Errorf("invalid UEFI loader file name '%s'", fileName)
	}
	if err != nil {
		return offset, err
	}
	/* offset 0x85C7C */
	_ = binary.Write(bytes.NewBuffer(valArray[0:0]), binary.LittleEndian, uint32(fileInfo.Size()))
	_, err = imgFile.Write(valArray[:])
	if err != nil {
		return offset, err
	}
	/* offset 0x85C80 */
	_, err = imgFile.Seek(0x180, io.SeekCurrent)
	if err != nil {
		return offset, err
	}
	/* offset 0x85E00 */
	b := make([]byte, 8192)
	for {
		var n int
		n, err = uefiLoader.Read(b)
		if err == io.EOF {
			break
		} else if err != nil {
			return offset, fmt.Errorf("cannot read UEFI loader file: %v", err)
		}
		_, err = imgFile.Write(b[:n])
		if err != nil {
			return offset, err
		}
	}
	return offset + uefiFSSize, nil
}

func writeBlobPadded(imgFile *os.File, blob []byte, withTrailer bool) error {
	blobSize := len(blob)
	_, err := imgFile.Write(blob)
	if err != nil {
		return err
	}
	trailer := [...]byte{0x55, 0xAA}
	var paddedSize int
	if withTrailer {
		paddedSize = sectorSize - len(trailer)
	} else {
		paddedSize = sectorSize
	}
	if blobSize < paddedSize {
		_, err = imgFile.Seek(int64(paddedSize-blobSize), io.SeekCurrent)
		if err != nil {
			return err
		}
		if withTrailer {
			_, err = imgFile.Write(trailer[:])
		}
	}
	return err
}

func writeMBR(imgFile *os.File, uefi bool) error {
	info, err := imgFile.Stat()
	if err != nil {
		return fmt.Errorf("cannot get size of file: %v", err)
	}
	mbr := make([]byte, sectorSize)
	var n int
	n, err = imgFile.ReadAt(mbr, 0)
	if (n != cap(mbr)) || (err != nil) {
		return fmt.Errorf("failed to read MBR: %v", err)
	}
	if (mbr[sectorSize-2] != 0x55) || (mbr[sectorSize-1] != 0xAA) {
		return fmt.Errorf("invalid MBR signature")
	}
	parts := sectorSize - 2 - 4*partitionEntrySize

	// FS region comes right before MBR partitions
	var fsRegionType uint32
	_ = binary.Read(bytes.NewBuffer(mbr[parts-4:parts]), binary.LittleEndian, &fsRegionType)
	var fsRegionLength uint64
	if fsRegionType == regionFilesystem {
		_ = binary.Read(bytes.NewBuffer(mbr[parts-12:parts-4]), binary.LittleEndian, &fsRegionLength)
	}

	fsOffset := sectorSize + fsRegionLength + klogDumpSize
	partNum := 0
	if uefi {
		writePartition(mbr, partNum, true, 0xEF, fsOffset, uefiFSSize)
		partNum++
		fsOffset += uefiFSSize
	}
	writePartition(mbr, partNum, true, 0x83, fsOffset, bootFSSize)
	partNum++
	fsOffset += bootFSSize
	writePartition(mbr, partNum, true, 0x83, fsOffset, uint64(info.Size())-fsOffset)

	// Write MBR
	n, err = imgFile.WriteAt(mbr, 0)
	if (n != cap(mbr)) || (err != nil) {
		return fmt.Errorf("failed to write MBR: %v", err)
	}

	return nil
}

func writePartition(mbr []byte, index int, active bool, pType uint8, offset uint64, size uint64) {
	partEntry := sectorSize - 2 - (4-index)*partitionEntrySize
	part := mbr[partEntry : partEntry+partitionEntrySize]
	if active {
		part[0] = 0x80
	} else {
		part[0] = 0x00
	}
	part[4] = pType
	mbrCHS(part[1:4], offset)
	mbrCHS(part[5:8], offset+size-sectorSize)
	_ = binary.Write(bytes.NewBuffer(part[8:8]), binary.LittleEndian, uint32(offset/sectorSize))
	_ = binary.Write(bytes.NewBuffer(part[12:12]), binary.LittleEndian, uint32(size/sectorSize))
}

func mbrCHS(chs []byte, offset uint64) {
	sectorOffset := offset / sectorSize
	cyl := (sectorOffset / sectorsPerTrack) / heads
	head := byte((sectorOffset / sectorsPerTrack) % heads)
	sec := (sectorOffset % sectorsPerTrack) + 1
	if cyl > maxCyl {
		cyl = maxCyl
		head = 254
		sec = 63
	}
	chs[0] = head
	chs[1] = byte((cyl >> 8) | sec)
	chs[2] = byte(cyl & 0xff)
}
