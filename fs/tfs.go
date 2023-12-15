package fs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const tfsMagic = "NVMTFS"
const tfsVersion = 5
const oldTfsVersion = 4

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
	typeBuffer  = 0
	typeTuple   = 1
	typeVector  = 2
	typeInteger = 3
	typeString  = 4
)

const symlinkHopsMax = 8

type tfs struct {
	imgFile     *os.File
	imgOffset   uint64
	size        uint64
	allocated   uint64
	uuid        [16]byte
	label       string
	currentExt  *tlogExt
	symDict     map[string]int
	nonSymCount int
	staging     []byte
	decoder     tfsDecoder
	root        *map[string]interface{}
}

func (t *tfs) logInit(oldEncoding bool) error {
	if (t.size != 0) && (t.allocated+sectorSize > t.size) {
		return fmt.Errorf("available space (%d bytes) too small, required %d", t.size-t.allocated, sectorSize)
	}
	t.currentExt = t.newLogExt(true, oldEncoding)
	t.allocated += sectorSize
	return t.logExtend()
}

func (t *tfs) newLogExt(initial bool, oldEncoding bool) *tlogExt {
	var extSize uint
	if initial {
		extSize = sectorSize
	} else {
		extSize = logExtensionSize
	}
	logExt := &tlogExt{
		offset:      t.allocated,
		oldEncoding: oldEncoding,
		buffer:      make([]byte, 0, extSize),
	}
	logExt.buffer = append(logExt.buffer, tfsMagic...)
	version := uint(tfsVersion)
	if oldEncoding {
		version = oldTfsVersion
	}
	logExt.buffer = appendVarint(logExt.buffer, version)
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
	logExt := t.newLogExt(false, t.currentExt.oldEncoding)
	t.currentExt.linkTo(logExt.offset)
	t.allocated += logExtensionSize
	err := t.currentExt.flush(t.imgFile, t.imgOffset)
	if err != nil {
		return err
	}
	t.currentExt = logExt
	return nil
}

func (t *tfs) readLogExt(offset, size uint64) (uint64, error) {
	buffer := make([]byte, size)
	if _, err := t.imgFile.ReadAt(buffer, int64(t.imgOffset+offset)); err != nil {
		return 0, fmt.Errorf("cannot read image file: %v", err)
	}
	if bytes.Compare(buffer[0:len(tfsMagic)], []byte(tfsMagic)) != 0 {
		return 0, errors.New("TFS magic number not found")
	}
	offset = uint64(len(tfsMagic))
	version, err := getVarint(buffer, &offset)
	if err != nil {
		return 0, err
	}
	var oldEncoding bool
	if version == 4 {
		oldEncoding = true
	} else if version == tfsVersion {
		oldEncoding = false
	} else {
		return 0, fmt.Errorf("TFS version mismatch: expected %d, found %d", tfsVersion, version)
	}
	_size, err := getVarint(buffer, &offset)
	if err != nil {
		return 0, err
	}
	if _size != uint(size/sectorSize) {
		return 0, fmt.Errorf("unexpected TFS log extension size %d, expected %d",
			_size, size/sectorSize)
	}
	if _size == 1 { // first log "extension"
		copy(t.uuid[:], buffer[offset:])
		offset += uint64(len(t.uuid))
		for _, c := range string(buffer[offset:]) {
			if c == 0 {
				break
			}
			t.label += string(c)
		}
		offset += uint64(len(t.label) + 1)
	}
	for {
		record := buffer[offset]
		offset++
		switch record {
		case endOfLog:
			return 0, nil
		case tupleAvailable:
			if t.decoder.tupleRemain > 0 {
				err = fmt.Errorf("unexpected tupleAvailable record (tupleRemain: %d)",
					t.decoder.tupleRemain)
				return 0, err
			}
			tupleTotalLen, err := getVarint(buffer, &offset)
			if err != nil {
				return 0, err
			}
			length, err := getVarint(buffer, &offset) // segment length
			if err != nil {
				return 0, err
			}
			if length > tupleTotalLen {
				err = fmt.Errorf("invalid tupleAvailable record (length: %d, total length: %d, "+
					"offset: %d)", length, tupleTotalLen, offset)
				return 0, err
			}
			if length == tupleTotalLen {
				var decoded uint64
				_, err := t.decodeValue(buffer[offset:offset+uint64(length)], &decoded, oldEncoding)
				if err != nil {
					return 0, err
				}
			} else {
				t.staging = make([]byte, length)
				copy(t.staging, buffer[offset:offset+uint64(length)])
				t.decoder.tupleRemain = tupleTotalLen - length
			}
			offset += uint64(length)
		case tupleExtended:
			length, err := getVarint(buffer, &offset)
			if err != nil {
				return 0, err
			}
			if length > t.decoder.tupleRemain {
				err = fmt.Errorf("invalid tupleExtended record (length: %d, tupleRemain: %d)",
					length, t.decoder.tupleRemain)
				return 0, err
			}
			t.staging = append(t.staging, buffer[offset:offset+uint64(length)]...)
			t.decoder.tupleRemain -= length
			if t.decoder.tupleRemain == 0 {
				var decoded uint64
				_, err := t.decodeValue(t.staging, &decoded, oldEncoding)
				if err != nil {
					return 0, err
				}
			}
			offset += uint64(length)
		case endOfSegment:
			continue
		case logExtensionLink:
			sector, err := getVarint(buffer, &offset)
			if err != nil {
				return 0, err
			}
			length, err := getVarint(buffer, &offset)
			if err != nil {
				return 0, err
			}
			if length*sectorSize != logExtensionSize {
				return 0, fmt.Errorf("logExtensionLink record with unexpected length %d", length)
			}
			return uint64(sector * sectorSize), nil
		default:
			return 0, fmt.Errorf("unknown record %d (offset %d)", record, offset)
		}
	}
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

func (t *tfs) encodeValue(value interface{}) error {
	str, isStr := value.(string)
	if isStr {
		strType := byte(typeString)
		if t.currentExt.oldEncoding {
			strType = typeBuffer
		}
		t.encodeString(str, strType)
		return nil
	}
	strSlice, isStrSlice := value.([]string)
	if isStrSlice {
		if !t.currentExt.oldEncoding {
			vector := make([]interface{}, len(strSlice))
			for i, str := range strSlice {
				vector[i] = str
			}
			return t.encodeVector(vector)
		}
		tuple := make(map[string]interface{})
		for i, str := range strSlice {
			tuple[strconv.Itoa(i)] = str
		}
		return t.encodeTuple(tuple)
	}
	slice, isSlice := value.([]interface{})
	if isSlice {
		if !t.currentExt.oldEncoding {
			return t.encodeVector(slice)
		}
		tuple := make(map[string]interface{})
		for i, val := range slice {
			tuple[strconv.Itoa(i)] = val
		}
		return t.encodeTuple(tuple)
	}
	tuple, isTuple := value.(map[string]interface{})
	if isTuple {
		return t.encodeTuple(tuple)
	}
	return fmt.Errorf("unknown type of value %p", value)
}

func (t *tfs) encodeMetadata(name string, value interface{}) error {
	t.encodeSymbol(name)
	err := t.encodeValue(value)
	return err
}

func (t *tfs) encodeSymbol(name string) {
	index := t.symDict[name]
	if index != 0 {
		t.pushHeader(entryReference, typeBuffer, index)
	} else {
		t.encodeString(name, typeBuffer)
		t.symDict[name] = 1 + len(t.symDict) + t.nonSymCount
	}
}

func (t *tfs) encodeTupleHeader(tupleEntries int) {
	t.pushHeader(entryImmediate, typeTuple, tupleEntries)
	t.nonSymCount++
}

func (t *tfs) encodeTuple(tuple map[string]interface{}) error {
	t.encodeTupleHeader(len(tuple))
	for k, v := range tuple {
		err := t.encodeMetadata(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tfs) encodeString(s string, dataType byte) {
	t.pushHeader(entryImmediate, dataType, len(s))
	t.staging = append(t.staging, s...)
}

func (t *tfs) encodeVector(vector []interface{}) error {
	t.pushHeader(entryImmediate, typeVector, len(vector))
	t.nonSymCount++
	for _, value := range vector {
		err := t.encodeValue(value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tfs) decodeValue(buffer []byte, offset *uint64, oldEncoding bool) (interface{}, error) {
	var entry, dataType byte
	length, err := getHeader(buffer, offset, &entry, &dataType, oldEncoding)
	if err != nil {
		return nil, err
	}
	switch dataType {
	case typeTuple:
		return t.decodeTuple(buffer, offset, entry, length, oldEncoding)
	case typeBuffer:
		return t.decodeBuf(buffer, offset, entry, length)
	case typeVector:
		return t.decodeVector(buffer, offset, entry, length, oldEncoding)
	case typeInteger:
		return t.decodeInteger(buffer, offset, entry, length)
	case typeString:
		return t.decodeBuf(buffer, offset, entry, length)
	default:
		return nil, fmt.Errorf("unknown data type %d (offset %d)", dataType, *offset)
	}
}

func (t *tfs) decodeVector(buffer []byte, offset *uint64, entry byte, length uint, oldEncoding bool) (*[]interface{}, error) {
	var vector *[]interface{}
	if entry == entryImmediate {
		newVector := make([]interface{}, length)
		vector = &newVector
		t.decoder.dict[len(t.decoder.dict)+1] = vector
	} else {
		ref, err := getVarint(buffer, offset)
		if err != nil {
			return nil, err
		}
		vector, err = t.getDictVector(ref)
		if err != nil {
			return vector, err
		}
	}
	for i := 0; i < int(length); i++ {
		value, err := t.decodeValue(buffer, offset, oldEncoding)
		if err != nil {
			return vector, err
		}
		(*vector)[i] = value
	}
	return vector, nil
}

func (t *tfs) decodeTuple(buffer []byte, offset *uint64, entry byte, length uint, oldEncoding bool) (*map[string]interface{}, error) {
	var tuple *map[string]interface{}
	if entry == entryImmediate {
		newTuple := make(map[string]interface{})
		tuple = &newTuple
		t.decoder.dict[len(t.decoder.dict)+1] = tuple
	} else {
		ref, err := getVarint(buffer, offset)
		if err != nil {
			return nil, err
		}
		tuple, err = t.getDictTuple(ref)
		if err != nil {
			return tuple, err
		}
	}
	for i := 0; i < int(length); i++ {
		var nameEntry, nameType byte
		n, err := getHeader(buffer, offset, &nameEntry, &nameType, oldEncoding)
		if err != nil {
			return tuple, err
		}
		if nameType != typeBuffer {
			return tuple, errors.New("unexpected name type for symbol")
		}
		symbol, err := t.decodeSymbol(buffer, offset, nameEntry, n)
		if err != nil {
			return tuple, err
		}
		value, err := t.decodeValue(buffer, offset, oldEncoding)
		if err != nil {
			return tuple, err
		}
		if symbol != "" { // ignore attributes with empty symbols created by older versions of ops
			if bufValue, ok := value.(string); ok && (bufValue == "") {
				// an empty buffer is used to delete an entry from a tuple
				delete(*tuple, symbol)
			} else {
				(*tuple)[symbol] = value
			}
		}
	}
	return tuple, nil
}

func (t *tfs) decodeSymbol(buffer []byte, offset *uint64, entry byte, length uint) (string, error) {
	if entry == entryImmediate {
		sym := string(buffer[*offset : *offset+uint64(length)])
		*offset += uint64(length)
		t.decoder.dict[len(t.decoder.dict)+1] = sym
		return sym, nil
	}
	return t.getDictString(length)
}

func (t *tfs) decodeBuf(buffer []byte, offset *uint64, entry byte, length uint) (string, error) {
	if length == 0 {
		return "", nil
	}
	if entry == entryImmediate {
		buf := string(buffer[*offset : *offset+uint64(length)])
		*offset += uint64(length)
		return buf, nil
	}
	return t.getDictString(length)
}

func (t *tfs) decodeInteger(buffer []byte, offset *uint64, entry byte, length uint) (string, error) {
	val, err := getVarint(buffer, offset)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(int64(val), 10), nil
}

func (t *tfs) getDictVector(ref uint) (*[]interface{}, error) {
	value := t.decoder.dict[int(ref)]
	if value == nil {
		return nil, fmt.Errorf("indirect vector %d not found", ref)
	}
	var isVector bool
	vector, isVector := value.(*[]interface{})
	if !isVector {
		return nil, fmt.Errorf("invalid indirect vector %d", ref)
	}
	return vector, nil
}

func (t *tfs) getDictTuple(ref uint) (*map[string]interface{}, error) {
	value := t.decoder.dict[int(ref)]
	if value == nil {
		return nil, fmt.Errorf("indirect tuple %d not found", ref)
	}
	var isTuple bool
	tuple, isTuple := value.(*map[string]interface{})
	if !isTuple {
		return nil, fmt.Errorf("invalid indirect tuple %d", ref)
	}
	return tuple, nil
}

func (t *tfs) getDictString(ref uint) (string, error) {
	value := t.decoder.dict[int(ref)]
	if value == nil {
		return "", fmt.Errorf("indirect string %d not found", ref)
	}
	str, isString := value.(string)
	if !isString {
		return "", fmt.Errorf("invalid indirect string %d", ref)
	}
	return str, nil
}

func (t *tfs) writeLink(name string, target string) error {
	tuple := make(map[string]interface{})
	tuple["linktarget"] = target
	return t.encodeMetadata(name, tuple)
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
	return t.encodeMetadata(name, tuple)
}

func (t *tfs) pushHeader(entry byte, dataType byte, length int) {
	len64 := uint64(length)
	bitCount := uint(64 - bits.LeadingZeros64(len64))
	var words uint
	immBits := uint(3)
	if t.currentExt.oldEncoding {
		immBits = 5
	}
	if bitCount > immBits {
		words = ((bitCount - immBits) + (7 - 1)) / 7
	}
	var first = (entry << 7) | (dataType << (immBits + 1)) |
		byte(len64>>(words*7))
	if words != 0 {
		first |= 1 << immBits
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

func (t *tfs) readExtent(p []byte, extent *map[string]interface{}, offset uint64) (int, uint64, error) {
	extentOffset, err := strconv.ParseUint(getString(extent, "offset"), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot parse extent offset: %v", err)
	}
	extentLength, err := strconv.ParseUint(getString(extent, "length"), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot parse extent length: %v", err)
	}
	extentOffset *= sectorSize
	extentLength *= sectorSize
	if extentLength-offset < uint64(len(p)) {
		p = p[:extentLength-offset]
	}
	n, err := t.imgFile.ReadAt(p, int64(t.imgOffset+extentOffset+offset))
	remain := extentLength - offset - uint64(n)
	return n, remain, err
}

func (t *tfs) stat(path string) (os.FileInfo, error) {
	tuple, parent, err := t.lookup(t.root, path)
	if err != nil {
		return nil, err
	}
	return &tfsFileInfo{
		tuple:       tuple,
		parentTuple: parent,
	}, nil
}

func (t *tfs) readLink(path string) (string, error) {
	tuple, _, err := t.lookup(t.root, path)
	if err != nil {
		return "", fmt.Errorf("cannot look up '%s': %v", path, err)
	}
	return getString(tuple, "linktarget"), nil
}

func (t *tfs) fileReader(path string) (io.Reader, error) {
	var fileTuple *map[string]interface{}
	currentDir := t.root
	hopCount := 0
	for {
		tuple, parent, err := t.lookup(currentDir, path)
		if err != nil {
			return nil, fmt.Errorf("cannot look up '%s': %v", path, err)
		}
		target := getString(tuple, "linktarget")
		if target == "" {
			fileTuple = tuple
			break
		}
		if hopCount == symlinkHopsMax {
			return nil, fmt.Errorf("too many symbolic links, aborting at '%s'", path)
		}
		hopCount++
		currentDir = parent
		path = target
	}
	extents := getTuple(fileTuple, "extents")
	if extents == nil {
		return nil, fmt.Errorf("'%s' is not a file", path)
	}
	var fileLength uint64 = 0
	fileLengthStr := getString(fileTuple, "filelength")
	if fileLengthStr != "" {
		var err error
		fileLength, err = strconv.ParseUint(fileLengthStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse file length for '%s': %v", path, err)
		}
	}
	var extentOffsets []int
	for fileOffset := range *extents {
		offset, err := strconv.ParseUint(fileOffset, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse file extent offset for '%s': %v", path, err)
		}
		extentOffsets = append(extentOffsets, int(offset*sectorSize))
	}
	sort.Ints(extentOffsets)
	return &tfsFileReader{
		t:             t,
		length:        fileLength,
		extents:       extents,
		extentOffsets: extentOffsets,
		offset:        0,
		currentExtent: nil,
	}, nil
}

func (t *tfs) readDir(path string) ([]os.FileInfo, error) {
	dir, _, err := t.lookup(t.root, path)
	if err != nil {
		return nil, err
	}
	children := getTuple(dir, "children")
	if children == nil {
		return nil, errors.New("not a directory")
	}
	var entries []os.FileInfo
	for k, v := range *children {
		if (k != ".") && (k != "..") {
			entries = append(entries, &tfsFileInfo{
				tuple:       v.(*map[string]interface{}),
				parentTuple: dir,
			})
		}
	}
	return entries, nil
}

func (t *tfs) getUUID() string {
	uuid := t.uuid

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

type tfsDecoder struct {
	tupleRemain uint
	dict        map[int]interface{}
}

type tfsFileInfo struct {
	tuple       *map[string]interface{}
	parentTuple *map[string]interface{}
}

func (i *tfsFileInfo) IsDir() bool {
	c := getTuple(i.tuple, "children")
	if c == nil {
		return false
	}
	return true
}

func (i *tfsFileInfo) ModTime() time.Time {
	c := getString(i.tuple, "mtime")
	var timestamp int64
	if c != "" {
		timestamp, _ = strconv.ParseInt(c, 10, 64)
	}
	return time.Unix(timestamp>>32, (timestamp&0xfffffffff)*0x10000000/1000000000)
}

func (i *tfsFileInfo) Mode() os.FileMode {
	if i.IsDir() {
		return os.ModeDir
	}
	if getString(i.tuple, "linktarget") != "" {
		return os.ModeSymlink
	}
	if getTuple(i.tuple, "extents") != nil {
		return 0 // regular file
	}
	return os.ModeIrregular
}

func (i *tfsFileInfo) Name() string {
	if i.parentTuple == i.tuple {
		return "/"
	}
	children := getTuple(i.parentTuple, "children")
	for k, v := range *children {
		if v == i.tuple {
			return k
		}
	}
	return "" // should never reach here
}

func (i *tfsFileInfo) Size() int64 {
	l := getString(i.tuple, "filelength")
	if l == "" {
		return 0 // non-regular file
	}
	length, _ := strconv.ParseInt(l, 10, 64)
	return length
}

func (i *tfsFileInfo) Sys() interface{} {
	return i.tuple
}

type tfsFileReader struct {
	t             *tfs
	length        uint64
	extents       *map[string]interface{}
	extentOffsets []int
	curExtIndex   int
	currentExtent *map[string]interface{}
	offset        uint64
}

func (r *tfsFileReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.length {
		return 0, io.EOF
	}
	if len(r.extentOffsets) == 0 {
		n = len(p)
		for i := 0; i < n; i++ {
			p[i] = 0
		}
		return n, nil
	}
	n = 0
	for {
		if r.currentExtent == nil {
			min := 0
			max := len(r.extentOffsets) - 1
			selected := -1
			for {
				selected = (min + max) / 2
				if min == max {
					break
				}
				if uint64(r.extentOffsets[selected]) > r.offset {
					max = selected - 1
					if max < min {
						max = min
					}
				} else if uint64(r.extentOffsets[selected+1]) <= r.offset {
					min = selected + 1
				} else {
					break
				}
			}
			if selected >= 0 {
				r.selectExtent(selected)
			}
		}
		var extentStart uint64
		var extentLength uint64
		if r.currentExtent == nil {
			extentStart = r.length
		} else {
			extentStart, extentLength, err = r.getExtentRange()
			if err != nil {
				break
			}
			if extentStart+extentLength <= r.offset {
				if r.curExtIndex < len(r.extentOffsets)-1 {
					r.selectExtent(r.curExtIndex + 1)
					extentStart, extentLength, err = r.getExtentRange()
					if err != nil {
						break
					}
				} else {
					r.currentExtent = nil
					extentStart = r.length
				}
			}
		}
		uninited := (r.currentExtent == nil) || (getString(r.currentExtent, "uninited") != "")
		if (r.currentExtent != nil) && uninited {
			extentStart += extentLength * sectorSize
		}
		if extentStart > r.offset {
			zeros := extentStart - r.offset
			if r.offset+zeros > r.length {
				zeros = r.length - r.offset
			}
			if zeros > uint64(len(p[n:])) {
				zeros = uint64(len(p[n:]))
			}
			for i := 0; uint64(i) < zeros; i++ {
				p[n+i] = 0
			}
			n += int(zeros)
			r.offset += zeros
		}
		if (r.currentExtent != nil) && !uninited {
			readCount := len(p) - n
			if r.offset+uint64(readCount) > r.length {
				readCount = int(r.length - r.offset)
			}
			dest := p[n : n+readCount]
			var remain uint64
			readCount, remain, err = r.t.readExtent(dest, r.currentExtent, r.offset-extentStart)
			n += readCount
			r.offset += uint64(readCount)
			if remain == 0 { // all data from the current extent has been read
				r.currentExtent = nil
			}
		}
		if (r.offset >= r.length) && (err == nil) {
			err = io.EOF
		}
		if (n == len(p)) || (err != nil) {
			break
		}
	}
	return n, err
}

func (r *tfsFileReader) getExtentRange() (uint64, uint64, error) {
	start := uint64(r.extentOffsets[r.curExtIndex])
	length, err := strconv.ParseUint(getString(r.currentExtent, "length"), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot parse extent length: %v", err)
	}
	return start, length * sectorSize, nil
}

func (r *tfsFileReader) selectExtent(index int) {
	r.curExtIndex = index
	ext := (*r.extents)[strconv.Itoa(r.extentOffsets[index]/sectorSize)]
	r.currentExtent = ext.(*map[string]interface{})
}

type tlogExt struct {
	offset      uint64
	oldEncoding bool
	buffer      []byte
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

func getHeader(buffer []byte, offset *uint64, entry *byte, dataType *byte, oldEncoding bool) (uint, error) {
	if int(*offset) >= len(buffer) {
		return 0, fmt.Errorf("getHeader(): buffer length %d exhausted", len(buffer))
	}
	b := buffer[*offset]
	*offset++
	*entry = b >> 7
	immBits := uint(3)
	if oldEncoding {
		immBits = 5
	}
	*dataType = (b >> (immBits + 1)) & ((1 << (6 - immBits)) - 1)
	length := uint(b & ((1 << immBits) - 1))
	if (b & (1 << immBits)) != 0 {
		for {
			if int(*offset) >= len(buffer) {
				return 0, fmt.Errorf("getHeader(): buffer length %d exhausted", len(buffer))
			}
			b := buffer[*offset]
			*offset++
			length = (length << 7) | uint(b&0x7f)
			if (b & 0x80) == 0 {
				break
			}
		}
	}
	return length, nil
}

func getVarint(buffer []byte, offset *uint64) (uint, error) {
	var result uint
	for {
		if int(*offset) >= len(buffer) {
			return 0, fmt.Errorf("getVarint(): buffer length %d exhausted", len(buffer))
		}
		b := buffer[*offset]
		*offset++
		result = (result << 7) | uint(b&0x7f)
		if (b & 0x80) == 0 {
			break
		}
	}
	return result, nil
}

func getTuple(parent *map[string]interface{}, child string) *map[string]interface{} {
	entry := (*parent)[child]
	if entry == nil {
		return nil
	}
	var isTuple bool
	tuple, isTuple := entry.(*map[string]interface{})
	if !isTuple {
		return nil
	}
	return tuple
}

func getString(parent *map[string]interface{}, child string) string {
	value := (*parent)[child]
	if value == nil {
		return ""
	}
	str, isString := value.(string)
	if !isString {
		return ""
	}
	return str
}

func getChild(parent *map[string]interface{}, child string) *map[string]interface{} {
	children := getTuple(parent, "children")
	if children == nil {
		return nil
	}
	return getTuple(children, child)
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
func tfsWrite(imgFile *os.File, imgOffset uint64, fsSize uint64, label string, root map[string]interface{}, oldEncoding bool) (*tfs, error) {
	tfs := newTfs(imgFile, imgOffset, fsSize)
	tfs.label = label
	rand.Seed(time.Now().UnixNano())
	_, err := rand.Read(tfs.uuid[:])
	if err != nil {
		return nil, fmt.Errorf("error generating random uuid: %v", err)
	}
	err = tfs.logInit(oldEncoding)
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
			err = tfs.encodeMetadata(k, v)
			if err != nil {
				return nil, err
			}
		}
	}
	return tfs, tfs.flush()
}

func tfsRead(imgFile *os.File, fsOffset, fsSize uint64) (*tfs, error) {
	tfs := newTfs(imgFile, fsOffset, fsSize)
	tfs.decoder.dict = make(map[int]interface{})
	nextExt, err := tfs.readLogExt(0, sectorSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read filesystem at first log extension: %v", err)
	}
	for nextExt != 0 {
		nextExt, err = tfs.readLogExt(nextExt, logExtensionSize)
		if err != nil {
			return nil, fmt.Errorf("cannot read filesystem log extension: %v", err)
		}
	}
	tfs.root, err = tfs.getDictTuple(1)
	if err != nil {
		return nil, err
	}
	fixupDirectory(tfs.root, tfs.root)
	return tfs, nil
}

func fixupDirectory(parent, dir *map[string]interface{}) {
	children := getTuple(dir, "children")
	if children == nil {
		return // not a directory
	}
	for _, child := range *children {
		fixupDirectory(dir, child.(*map[string]interface{}))
	}
	(*children)["."] = dir
	(*children)[".."] = parent
}

func (t *tfs) lookup(cwd *map[string]interface{}, path string) (*map[string]interface{}, *map[string]interface{}, error) {
	if strings.HasPrefix(path, "/") {
		cwd = t.root
	}
	tuple := cwd
	parent := cwd
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parent = tuple
			tuple = getChild(tuple, part)
			if tuple == nil {
				return nil, nil, os.ErrNotExist
			}
		}
	}
	if getTuple(tuple, "children") != nil {
		parent = getChild(tuple, "..")
	} else if strings.HasSuffix(path, "/") {
		return nil, nil, errors.New("not a directory")
	}
	return tuple, parent, nil
}

func (t *tfs) getTuple(name string) *map[string]interface{} {
	return getTuple(t.root, name)
}
