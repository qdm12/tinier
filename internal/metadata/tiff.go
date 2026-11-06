package metadata

import (
	"encoding/binary"
	"fmt"
)

const (
	// exifIdentifier starts the payload of the EXIF segment, before the TIFF
	// structure which holds the tags.
	exifIdentifier = "Exif\x00\x00"

	// tiffMagic is the number which tells the byte order of a TIFF structure.
	tiffMagic uint16 = 42
	// tiffHeaderSize is the size of the header of a TIFF structure: the byte
	// order, the magic number and the offset of the first image file directory.
	tiffHeaderSize = 8
	// ifdCountSize is the size of the field counting the entries of an image file
	// directory, and ifdNextSize is the size of the field holding the offset of
	// the next directory.
	ifdCountSize = 2
	ifdNextSize  = 4
	// ifdEntrySize is the size of an entry of an image file directory.
	ifdEntrySize = 12
	// maximum values held by the entries of an image file directory.
	maxByteValue  = 0xFF
	maxShortValue = 0xFFFF
	// sizes in bytes of the values of the entries of an image file directory.
	byteTypeSize  = 1
	shortTypeSize = 2
	longTypeSize  = 4
	// entryValueOffset is the offset, in an entry, of the 4 bytes holding its
	// value, or the offset of its value.
	entryValueOffset = 8

	// Tags holding the dimensions and the orientation of the image.
	imageWidthTag        uint16 = 0x0100
	imageLengthTag       uint16 = 0x0101
	orientationTag       uint16 = 0x0112
	exifSubIFDPointerTag uint16 = 0x8769
	pixelXDimensionTag   uint16 = 0xA002
	pixelYDimensionTag   uint16 = 0xA003

	// Types of the values of the entries of an image file directory.
	typeByte      uint16 = 1
	typeShort     uint16 = 3
	typeLong      uint16 = 4
	typeUndefined uint16 = 7

	// orientationNormal is the value of the orientation tag of an image which
	// is neither rotated nor mirrored.
	orientationNormal uint32 = 1
)

// tiff reads and updates, in place, the entries of the image file directories
// of the TIFF structure an EXIF payload is made of.
type tiff struct {
	data  []byte
	order binary.ByteOrder
}

// newTiff returns the TIFF structure held by the given data, which is the
// payload of an EXIF segment, minus its identifier.
func newTiff(data []byte) (*tiff, error) {
	if len(data) < tiffHeaderSize {
		return nil, fmt.Errorf("%w: the tiff structure is truncated", ErrInvalidExif)
	}

	var order binary.ByteOrder
	switch {
	case data[0] == 'I' && data[1] == 'I':
		order = binary.LittleEndian
	case data[0] == 'M' && data[1] == 'M':
		order = binary.BigEndian
	default:
		return nil, fmt.Errorf("%w: unknown byte order %q", ErrInvalidExif, data[:2])
	}

	if order.Uint16(data[2:4]) != tiffMagic {
		return nil, fmt.Errorf("%w: unexpected magic number %d", ErrInvalidExif, order.Uint16(data[2:4]))
	}

	return &tiff{data: data, order: order}, nil
}

// firstIFDOffset returns the offset of the first image file directory.
func (t *tiff) firstIFDOffset() int {
	return int(t.order.Uint32(t.data[4:8]))
}

// entry returns the entry of the given tag, in the image file directory at the
// given offset, and reports whether it was found.
func (t *tiff) entry(ifdOffset int, tag uint16) (ifdEntry, bool) {
	entries, ok := t.entries(ifdOffset)
	if !ok {
		return ifdEntry{}, false
	}

	for _, e := range entries {
		if e.tag == tag {
			return e, true
		}
	}

	return ifdEntry{}, false
}

// entries returns the entries of the image file directory at the given offset,
// and reports whether they could be read.
func (t *tiff) entries(ifdOffset int) ([]ifdEntry, bool) {
	if ifdOffset < tiffHeaderSize || ifdOffset+ifdCountSize > len(t.data) {
		return nil, false
	}

	count := int(t.order.Uint16(t.data[ifdOffset : ifdOffset+ifdCountSize]))
	firstEntry := ifdOffset + ifdCountSize
	if firstEntry+count*ifdEntrySize+ifdNextSize > len(t.data) {
		return nil, false
	}

	entries := make([]ifdEntry, count)
	for i := range entries {
		offset := firstEntry + i*ifdEntrySize
		entries[i] = ifdEntry{
			tag:         t.order.Uint16(t.data[offset : offset+2]),
			dataType:    t.order.Uint16(t.data[offset+2 : offset+4]),
			count:       t.order.Uint32(t.data[offset+4 : offset+8]),
			valueOffset: offset + entryValueOffset,
		}
	}

	return entries, true
}

// subIFDOffset returns the offset of the image file directory pointed to by the
// given tag, and reports whether it was found.
func (t *tiff) subIFDOffset(ifdOffset int, tag uint16) (int, bool) {
	e, ok := t.entry(ifdOffset, tag)
	if !ok {
		return 0, false
	}

	offset, ok := t.uint(e)
	if !ok || int(offset) >= len(t.data) {
		return 0, false
	}

	return int(offset), true
}

// ifdEntry is an entry of an image file directory of a TIFF structure.
type ifdEntry struct {
	tag      uint16
	dataType uint16
	count    uint32
	// valueOffset is the offset, in the data of the TIFF structure, of the 4
	// bytes holding the value of the entry, or the offset of its value.
	valueOffset int
}

// uint returns the value of an entry holding a single integer, and reports
// whether it could be read.
func (t *tiff) uint(e ifdEntry) (uint32, bool) {
	size, ok := typeSize(e.dataType)
	if !ok {
		return 0, false
	}

	value, ok := t.valueBytes(e, size)
	if !ok {
		return 0, false
	}

	switch size {
	case byteTypeSize:
		return uint32(value[0]), true
	case shortTypeSize:
		return uint32(t.order.Uint16(value)), true
	default:
		return t.order.Uint32(value), true
	}
}

// typeSize returns the size in bytes of a value of the given type, and reports
// whether the type holds a single integer.
func typeSize(dataType uint16) (size int, ok bool) {
	switch dataType {
	case typeByte, typeUndefined:
		return byteTypeSize, true
	case typeShort:
		return shortTypeSize, true
	case typeLong:
		return longTypeSize, true
	default:
		return 0, false
	}
}

// valueBytes returns the bytes of the value of an entry of the given size, held
// at the offset of the value, and reports whether they can be read. The entry
// must hold a single value.
func (t *tiff) valueBytes(e ifdEntry, size int) ([]byte, bool) {
	if e.count != 1 || e.valueOffset < 0 ||
		e.valueOffset+size > len(t.data) {
		return nil, false
	}

	return t.data[e.valueOffset : e.valueOffset+size], true
}

// setUint updates, in place, the value of an entry holding a single integer.
// It does nothing if the entry holds another kind of value, or if the value
// would not fit the type of the entry.
func (t *tiff) setUint(e ifdEntry, value uint32) {
	size, ok := typeSize(e.dataType)
	if !ok {
		return
	}

	rawValue, ok := t.valueBytes(e, size)
	if !ok {
		return
	}

	switch size {
	case byteTypeSize:
		if value > maxByteValue {
			return
		}
		rawValue[0] = byte(value)
	case shortTypeSize:
		if value > maxShortValue {
			return
		}
		t.order.PutUint16(rawValue, uint16(value))
	default:
		t.order.PutUint32(rawValue, value)
	}
}

// setOrientationNormal sets the orientation tag to the value describing an
// image which is neither rotated nor mirrored. It does nothing when the tag is
// absent, an absent tag meaning a normal orientation.
func (t *tiff) setOrientationNormal() {
	e, ok := t.entry(t.firstIFDOffset(), orientationTag)
	if !ok {
		return
	}

	t.setUint(e, orientationNormal)
}

// setDimensions updates the tags holding the dimensions of the image, when they
// are present, in the first image file directory and in the one pointed to by
// the EXIF sub-IFD pointer.
func (t *tiff) setDimensions(width, height uint32) {
	if width == 0 || height == 0 {
		return
	}

	t.setEntryUint(t.firstIFDOffset(), imageWidthTag, width)
	t.setEntryUint(t.firstIFDOffset(), imageLengthTag, height)

	subIFDOffset, ok := t.subIFDOffset(t.firstIFDOffset(), exifSubIFDPointerTag)
	if !ok {
		return
	}

	t.setEntryUint(subIFDOffset, pixelXDimensionTag, width)
	t.setEntryUint(subIFDOffset, pixelYDimensionTag, height)
}

// setEntryUint updates the value of the entry of the given tag, when the entry
// is present and holds a single integer.
func (t *tiff) setEntryUint(ifdOffset int, tag uint16, value uint32) {
	e, ok := t.entry(ifdOffset, tag)
	if !ok {
		return
	}

	t.setUint(e, value)
}
