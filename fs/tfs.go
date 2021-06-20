package fs

import (
	"fmt"
	"io"
	"math/bits"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const tfsMagic = "NVMTFS"
const tfsVersion = 4

const logExtensionSize = 1024 * sectorSize

const (
	endOfLog         byte = 1
	tupleAvailable   byte = 2
	tupleExtended    byte = 3
	endOfSegment     byte = 4
	logExtensionLink byte = 5
)

const maxVarintSize = 10 // max(pad(64, 7) / 7)
const tupleAvailableHeaderSize = 1 + 2*maxVarintSize
const tupleAvailableMinSize = tupleAvailableHeaderSize + 32 // arbitrary
const tfsExtLinkBytes = 1 + 2*maxVarintSize

const (
	entryReference = 0
	entryImmediate = 1
)

const (
	typeBuffer = 0
	typeTuple  = 1
)

type tfs struct {
	imgFile    *os.File
	imgOffset  uint64
	size       uint64
	allocated  uint64
	uuid       [16]byte
	label      string
	currentExt *tlogExt
	symDict    map[string]int
	tupleCount int
	staging    []byte
}

func (t *tfs) logInit() error {
	if (t.size != 0) && (t.allocated+sectorSize > t.size) {
		return fmt.Errorf("available space (%d bytes) too small, required %d", t.size-t.allocated, sectorSize)
	}
	t.currentExt = t.newLogExt(true)
	t.allocated += sectorSize
	return t.logExtend()
}

func (t *tfs) newLogExt(initial bool) *tlogExt {
	var extSize uint
	if initial {
		extSize = sectorSize
	} else {
		extSize = logExtensionSize
	}
	logExt := &tlogExt{
		offset: t.allocated,
		buffer: make([]byte, 0, extSize),
	}
	logExt.buffer = append(logExt.buffer, tfsMagic...)
	logExt.buffer = appendVarint(logExt.buffer, tfsVersion)
	logExt.buffer = appendVarint(logExt.buffer, extSize/sectorSize)
	if initial {
		logExt.buffer = append(logExt.buffer, t.uuid[:]...)
		logExt.buffer = append(logExt.buffer, t.label...)
		logExt.buffer = append(logExt.buffer, byte(0x00)) // label string terminator
	}
	return logExt
}

func (t *tfs) logExtend() error {
	if (t.size != 0) && (t.allocated+logExtensionSize > t.size) {
		return fmt.Errorf("available space (%d bytes) too small, required %d", t.size-t.allocated, logExtensionSize)
	}
	logExt := t.newLogExt(false)
	t.currentExt.linkTo(logExt.offset)
	t.allocated += logExtensionSize
	err := t.currentExt.flush(t.imgFile, t.imgOffset)
	if err != nil {
		return err
	}
	t.currentExt = logExt
	return nil
}

func (t *tfs) writeDirEntries(dir map[string]interface{}) error {
	var err error
	t.encodeSymbol("children")
	t.encodeTupleHeader(len(dir))
	for k, v := range dir {
		nvalue, nok := v.(link)
		if nok {
			err = t.writeLink(k, nvalue.path)
			if err != nil {
				return err
			}
			continue
		}
		value, ok := v.(string)
		if ok {
			err = t.writeFile(k, value)
		} else {
			t.encodeSymbol(k)
			t.encodeTupleHeader(1) // for "children" attribute
			err = t.writeDirEntries(v.(map[string]interface{}))
		}
		if err != nil {
			break
		}
	}
	return err
}

func (t *tfs) encodeMetadata(name string, value interface{}) {
	t.encodeSymbol(name)
	str, isStr := value.(string)
	if isStr {
		t.encodeString(str)
		return
	}
	strSlice, isStrSlice := value.([]string)
	if isStrSlice {
		tuple := make(map[string]interface{})
		for i, str := range strSlice {
			tuple[strconv.Itoa(i)] = str
		}
		t.encodeTuple(tuple)
		return
	}
	slice, isSlice := value.([]interface{})
	if isSlice {
		tuple := make(map[string]interface{})
		for i, val := range slice {
			tuple[strconv.Itoa(i)] = val
		}
		t.encodeTuple(tuple)
		return
	}
	t.encodeTuple(value.(map[string]interface{}))
}

func (t *tfs) encodeSymbol(name string) {
	index := t.symDict[name]
	if index != 0 {
		t.pushHeader(entryReference, typeBuffer, index)
	} else {
		t.encodeString(name)
		t.symDict[name] = 1 + len(t.symDict) + t.tupleCount
	}
}

func (t *tfs) encodeTupleHeader(tupleEntries int) {
	t.pushHeader(entryImmediate, typeTuple, tupleEntries)
	t.tupleCount++
}

func (t *tfs) encodeTuple(tuple map[string]interface{}) {
	t.encodeTupleHeader(len(tuple))
	for k, v := range tuple {
		t.encodeMetadata(k, v)
	}
}

func (t *tfs) encodeString(s string) {
	t.pushHeader(entryImmediate, typeBuffer, len(s))
	t.staging = append(t.staging, s...)
}

func (t *tfs) writeLink(name string, target string) error {
	tuple := make(map[string]interface{})
	tuple["linktarget"] = target
	t.encodeMetadata(name, tuple)
	return nil
}

func (t *tfs) writeFile(name string, hostPath string) error {
	file, err := os.Open(hostPath)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %v", hostPath, err)
	}
	defer file.Close()
	var info os.FileInfo
	info, err = file.Stat()
	if err != nil {
		return fmt.Errorf("cannot get size of file %s: %v", hostPath, err)
	}
	tuple := make(map[string]interface{})
	tuple["filelength"] = strconv.FormatInt(info.Size(), 10)
	extents := make(map[string]interface{})
	if info.Size() > 0 {
		sectors := uint64((info.Size() + sectorSize - 1) / sectorSize)
		paddedLen := sectors * sectorSize
		if (t.size != 0) && (t.allocated+paddedLen > t.size) {
			return fmt.Errorf("available space (%d bytes) too small, required %d", t.size-t.allocated, paddedLen)
		}
		_, err = t.imgFile.Seek(int64(t.imgOffset+t.allocated), 0)
		if err != nil {
			return fmt.Errorf("cannot seek image file: %v", err)
		}
		b := make([]byte, 8192)
		for {
			var n int
			n, err = file.Read(b)
			if err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("cannot read file %s: %v", hostPath, err)
			}
			n, err = t.imgFile.Write(b[:n])
			if err != nil {
				return fmt.Errorf("cannot write image file: %v", err)
			}
		}
		extent := make(map[string]interface{})
		extent["length"] = strconv.FormatUint(sectors, 10)
		extent["offset"] = strconv.FormatUint(t.allocated/sectorSize, 10)
		extent["allocated"] = extent["length"]
		t.allocated += paddedLen
		extents["0"] = extent
	}
	tuple["extents"] = extents
	t.encodeMetadata(name, tuple)
	return nil
}

func (t *tfs) pushHeader(entry byte, dataType byte, length int) {
	len64 := uint64(length)
	bitCount := uint(64 - bits.LeadingZeros64(len64))
	var words uint
	if bitCount > 5 {
		words = ((bitCount - 5) + (7 - 1)) / 7
	}
	var first = (entry << 7) | (dataType << 6) | byte(len64>>(words*7))
	if words != 0 {
		first |= 1 << 5
	}
	t.staging = append(t.staging, first)
	i := words
	for i > 0 {
		i--
		v := byte((len64 >> (i * 7)) & 0x7f)
		if i != 0 {
			v |= 0x80
		}
		t.staging = append(t.staging, v)
	}
}

func (t *tfs) flush() error {
	ext := t.currentExt
	written := 0
	for len(t.staging) > 0 {
		min := tupleAvailableMinSize + tfsExtLinkBytes
		available := cap(ext.buffer) - len(ext.buffer)
		if available < min {
			err := t.logExtend()
			if err != nil {
				return err
			}
			ext = t.currentExt
			available = cap(ext.buffer) - len(ext.buffer)
		}
		length := available - tfsExtLinkBytes - tupleAvailableHeaderSize
		if length > len(t.staging) {
			length = len(t.staging)
		}
		if written == 0 {
			ext.buffer = append(ext.buffer, tupleAvailable)
			ext.buffer = appendVarint(ext.buffer, uint(len(t.staging)))
		} else {
			ext.buffer = append(ext.buffer, tupleExtended)
		}
		ext.buffer = appendVarint(ext.buffer, uint(length))
		ext.buffer = append(ext.buffer, t.staging[:length]...)
		t.staging = t.staging[length:]
		written += length
	}
	ext.buffer = append(ext.buffer, endOfLog)
	err := ext.flush(t.imgFile, t.imgOffset)
	if err != nil {
		return err
	}
	var info os.FileInfo
	info, err = t.imgFile.Stat()
	if err != nil {
		return fmt.Errorf("cannot get size of image file: %v", err)
	}
	minSize := t.imgOffset
	if t.size != 0 {
		minSize += t.size
	} else {
		minSize += t.allocated
	}
	if uint64(info.Size()) < minSize {
		err = t.imgFile.Truncate(int64(minSize))
		if err != nil {
			return fmt.Errorf("cannot truncate image file: %v", err)
		}
	}
	return nil
}

type tlogExt struct {
	offset uint64
	buffer []byte
}

func (e *tlogExt) linkTo(extOffset uint64) {
	e.buffer = append(e.buffer, logExtensionLink)
	e.buffer = appendVarint(e.buffer, uint(extOffset/sectorSize))
	e.buffer = appendVarint(e.buffer, logExtensionSize/sectorSize)
}

func (e *tlogExt) flush(imgFile *os.File, imgOffset uint64) error {
	n, err := imgFile.WriteAt(e.buffer[:cap(e.buffer)], int64(imgOffset+e.offset))
	if err != nil {
		return err
	}
	if n != cap(e.buffer) {
		return fmt.Errorf("wrote %d out of %d bytes", n, cap(e.buffer))
	}
	return nil
}

func appendVarint(buffer []byte, x uint) []byte {
	last := 0
	var tmp [maxVarintSize]byte
	tmp[0] = byte(x & 0x7f)
	x >>= 7
	for x != 0 {
		last++
		tmp[last] = byte(0x80 | (x & 0x7f))
		x >>= 7
	}
	for i := last; i >= 0; i-- {
		buffer = append(buffer, tmp[i])
	}
	return buffer
}

func newTfs(imgFile *os.File, imgOffset uint64, fsSize uint64) *tfs {
	return &tfs{
		imgFile:   imgFile,
		imgOffset: imgOffset,
		size:      fsSize,
		symDict:   make(map[string]int),
	}
}

// tfsWrite writes filesystem metadata and contents to image file
func tfsWrite(imgFile *os.File, imgOffset uint64, fsSize uint64, label string, root map[string]interface{}) (*tfs, error) {
	tfs := newTfs(imgFile, imgOffset, fsSize)
	tfs.label = label
	rand.Seed(time.Now().UnixNano())
	_, err := rand.Read(tfs.uuid[:])
	err = tfs.logInit()
	if err != nil {
		return nil, fmt.Errorf("cannot create filesystem log: %v", err)
	}
	tfs.encodeTupleHeader(len(root))
	for k, v := range root {
		if k == "children" {
			err = tfs.writeDirEntries(v.(map[string]interface{}))
			if err != nil {
				return nil, err
			}
		} else {
			tfs.encodeMetadata(k, v)
		}
	}
	return tfs, tfs.flush()
}
