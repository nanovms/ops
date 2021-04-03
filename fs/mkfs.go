package fs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

const (
	partitionBootFS = 0
	partitionRootFS = 1
)

const klogDumpSize = 4 * 1024
const bootFSSize = 12 * 1024 * 1024

// MkfsCommand wraps mkfs calls
type MkfsCommand struct {
	bootPath string
	label    string
	manifest *Manifest
	size     int64
	outPath  string
	rootTfs  *tfs
}

// NewMkfsCommand returns an instance of MkfsCommand
func NewMkfsCommand(m *Manifest) *MkfsCommand {
	return &MkfsCommand{
		bootPath: "",
		label:    "",
		manifest: m,
		size:     0,
		outPath:  "",
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
	}
	manifest := m.manifest
	var root map[string]interface{}
	if manifest != nil {
		manifest.finalize()
		if manifest.boot != nil {
			outOffset += klogDumpSize
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
	if m.bootPath != "" {
		err = writeMBR(outFile)
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

func writeMBR(imgFile *os.File) error {
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
	if fsRegionType != regionFilesystem {
		return fmt.Errorf("invalid boot record (missing filesystem region)")
	}
	var fsRegionLength uint64
	_ = binary.Read(bytes.NewBuffer(mbr[parts-12:parts-4]), binary.LittleEndian, &fsRegionLength)

	fsOffset := sectorSize + fsRegionLength + klogDumpSize
	writePartition(mbr, partitionBootFS, true, 0x83, fsOffset, bootFSSize)
	fsOffset += bootFSSize
	writePartition(mbr, partitionRootFS, true, 0x83, fsOffset, uint64(info.Size())-fsOffset)

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
