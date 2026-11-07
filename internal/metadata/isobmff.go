package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"slices"
)

// Names of the boxes of a base media file of the International Organization for
// Standardization (ISO 14496-12), the container of the AVIF, HEIF and MP4
// formats. Only the boxes holding the metadata tinier copies are named here.
const (
	fileTypeBoxName          = "ftyp"
	metaBoxName              = "meta"
	itemInfoBoxName          = "iinf"
	itemLocationBoxName      = "iloc"
	colorInfoBoxName         = "colr"
	itemPropertiesBoxName    = "iprp"
	propertyContainerBoxName = "ipco"
	itemDefinitionBoxName    = "infe"
)

// Names of the items of the metadata tinier copies, each one four characters
// long, the name of the XMP packet padded with a space. The other items, of the
// pixels, of the thumbnails and of the properties of the images, have no
// equivalent in a JPEG file, or are the pixels themselves.
const (
	exifItemName = "Exif"
	xmpItemName  = "Xmp "
)

// Names of the types of color information an item can carry. The first two
// describe the colors by parameters, where the primaries and the transfer
// characteristics are written down; the last two hold an ICC profile, and are
// the two the writers of AVIF files use. Any other name is read as holding a
// profile too, which the format of the box allows.
const (
	colorParametersName      = "nclx"
	colorLimitedRangeName    = "nslt"
	iccColorProfileName      = "prof"
	restrictedICCProfileName = "ricc"
)

// Brands of the AVIF files, which the box telling the type of a file holds at
// the start of its contents. The other brands of the same container, the ones
// of the HEIF and MP4 files, are not the ones tinier converts.
//
//nolint:gochecknoglobals // set of fixed brands, never mutated
var avifBrands = []string{"avif", "av01", "avis"}

const (
	// Sizes of the fields of the header of a box: the length and the name
	// precede its contents, and the length of a box longer than 4 GB is held by
	// eight bytes which follow the name.
	boxLengthSize        = 4
	boxNameSize          = 4
	longBoxLengthSize    = 8
	versionAndFlagsSize  = 4
	itemProtectionSize   = 2
	itemReferenceSize    = 2
	itemExtentCountSize  = 2
	maxMetadataBoxLength = 1 << 24
	// Sizes of the fields of a box which its version sets: two bytes in the
	// first versions of a box, four in the following ones.
	shortFieldSize = 2
	longFieldSize  = 4
	// Version of a box of an item definition, and of the box of the item
	// location, from which an identifier is held by four bytes.
	wideIdentifierVersion = 3
	// Version of the box of the item location from which every piece of an item
	// carries the index of the file it is part of.
	indexedExtentsVersion = 2
	// Shift and mask of one of the two sizes packed, four bits by four bits, in a
	// byte of the box of the item location.
	packedSizeShift = 4
	packedSizeMask  = 0xF
	// Largest number of bytes a field of a base media file is held by, which
	// bounds the reads of the cursor.
	maxFieldSize = 8
)

// isAVIF tells if the bytes at the start of a file are the ones of an AVIF
// file: the length and the name of the first box, the one telling the type of
// the file, then the brand of the file itself.
func isAVIF(start []byte) bool {
	const brandOffset = boxLengthSize + boxNameSize

	if len(start) < brandOffset+boxNameSize {
		return false
	}
	if string(start[boxLengthSize:brandOffset]) != fileTypeBoxName {
		return false
	}

	return slices.Contains(avifBrands, string(start[brandOffset:brandOffset+boxNameSize]))
}

// avifMetadataSegments returns the metadata of an AVIF file as the segments of
// the JPEG file which would hold the same metadata. An AVIF file keeps its
// metadata in items, pieces of the file located by offsets by the box of the
// item location and named by the box of the item information: the EXIF
// structure, the XMP packet, and the profile of the colors, held by the box of
// the properties of the items.
//
// Only the metadata a JPEG file has an equivalent of is returned. The items of
// the pixels, of the properties of the items and of the media descriptions have
// no equivalent, and so is skipped any box tinier cannot read, which leaves the
// conversion and the metadata of the other items go on.
func avifMetadataSegments(reader io.ReaderAt, fileSize int64) ([]segment, error) {
	metaContents, found, err := topLevelBoxContents(reader, fileSize, metaBoxName)
	if err != nil || !found {
		return nil, err
	}

	// The contents of a box which has a version, as the box of the metadata has,
	// start with its version and its flags.
	if len(metaContents) < versionAndFlagsSize {
		return nil, truncatedErr(nil)
	}
	boxes, err := containedBoxes(metaContents[versionAndFlagsSize:])
	if err != nil {
		return nil, err
	}

	definitions, err := itemDefinitions(boxContents(boxes, itemInfoBoxName))
	if err != nil {
		return nil, err
	}
	locations, err := itemLocations(boxContents(boxes, itemLocationBoxName))
	if err != nil {
		return nil, err
	}

	segments := itemSegments(reader, definitions, locations)

	if profile := itemColorProfile(boxContents(boxes, itemPropertiesBoxName)); profile != nil {
		segments = append(segments, iccProfileSegment(profile))
	}

	return segments, nil
}

// itemSegments returns the segments holding the metadata of the items which are
// held by the file.
func itemSegments(reader io.ReaderAt, definitions []item,
	locations map[uint32]place) []segment {
	var segments []segment

	for _, definition := range definitions {
		where, found := locations[definition.identifier]
		if !found {
			continue
		}

		s, err := itemSegment(reader, definition, where)
		if err != nil {
			// An item which cannot be read leaves the other items, and the
			// conversion, go on.
			continue
		}
		if s.marker != 0 {
			segments = append(segments, s)
		}
	}

	return segments
}

// item is an item of an AVIF file, named by its identifier and by its type: the
// metadata tinier copies is an item of the EXIF or of the XMP type.
type item struct {
	identifier uint32
	name       string
}

// itemDefinitions returns the definitions of the items of the box of the item
// information of an AVIF file, in the order the file holds them, and an empty
// list when there is no such box.
func itemDefinitions(contents []byte) ([]item, error) {
	if len(contents) == 0 {
		return []item{}, nil
	}

	cursor := &cursor{contents: contents}
	version := cursor.uint(1)
	// The first version of the box counts its items on two bytes, the next ones
	// on four.
	entryCountSize := longFieldSize
	if version == 0 {
		entryCountSize = shortFieldSize
	}
	cursor.skip(versionAndFlagsSize - 1)
	definitionCount := int(cursor.uint(entryCountSize))
	if cursor.err != nil {
		return nil, truncatedErr(cursor.err)
	}

	entries, err := containedBoxes(contents[cursor.position:])
	if err != nil {
		return nil, err
	}

	definitions := make([]item, 0, definitionCount)

	for _, entry := range entries {
		if entry.name != itemDefinitionBoxName {
			continue
		}

		definition, held := itemDefinition(entry.contents)
		if held {
			definitions = append(definitions, definition)
		}
	}

	return definitions, nil
}

// itemDefinition returns the definition held by the contents of a box of an item
// definition, and tells if it holds one. An item of the pixels, of a thumbnail
// or of a referenced item is of no use to tinier, and so is the first version of
// the box, which names no type of item at all.
func itemDefinition(contents []byte) (definition item, held bool) {
	// The version, the identifier, the protection index and the name of the type
	// of the item have to be there for the box to define metadata.
	if len(contents) < versionAndFlagsSize+shortFieldSize+itemProtectionSize+boxNameSize {
		return item{}, false
	}

	cursor := &cursor{contents: contents}
	version := cursor.uint(1)
	// Items are named by their version: the identifier of an item is held by
	// four bytes since the third version of the box, and by two before.
	if version == 0 {
		return item{}, false
	}
	identifierSize := shortFieldSize
	if version >= wideIdentifierVersion {
		identifierSize = longFieldSize
	}
	cursor.skip(versionAndFlagsSize - 1)

	definition = item{
		identifier: cursor.uint(identifierSize),
	}
	cursor.skip(itemProtectionSize)
	definition.name = cursor.name()

	return definition, cursor.err == nil
}

// place is a piece of an AVIF file: where it is held, and how long it is.
type place struct {
	offset int64
	length int64
}

// itemLocations returns where each item of the box of the item location of an
// AVIF file is held, by identifier of the item, and an empty map when there is
// no such box.
//
// An item can be held by as many pieces as its writer needs, which the tiled
// images make use of. The metadata of an AVIF file is held by a single piece, so
// only the first one is kept.
func itemLocations(contents []byte) (map[uint32]place, error) {
	if len(contents) < versionAndFlagsSize+shortFieldSize {
		return map[uint32]place{}, nil
	}

	cursor := &cursor{contents: contents}
	version := cursor.uint(1)
	cursor.position = versionAndFlagsSize
	// The sizes of the fields which follow are packed, four bits by four bits,
	// in the bytes which follow the version and the flags of the box.
	sizes := cursor.uint(1)
	offsetSize, lengthSize := int(sizes>>packedSizeShift), int(sizes&packedSizeMask)
	sizes = cursor.uint(1)
	baseOffsetSize := int(sizes >> packedSizeShift)
	indexSize := int(sizes & packedSizeMask)
	// The identifier of an item, and the number of items, are held by four bytes
	// since the third version of the box, and by two before.
	itemCountSize, identifierSize := shortFieldSize, shortFieldSize
	if version >= wideIdentifierVersion {
		itemCountSize, identifierSize = longFieldSize, longFieldSize
	}
	itemCount := int(cursor.uint(itemCountSize))

	places := make(map[uint32]place, itemCount)

	for i := 0; i < itemCount; i++ {
		identifier := cursor.uint(identifierSize)
		cursor.skip(itemReferenceSize)
		baseOffset := cursor.long(baseOffsetSize)
		extentCount := int(cursor.uint(itemExtentCountSize))

		for extent := 0; extent < extentCount; extent++ {
			if version >= indexedExtentsVersion {
				// Index, in the file holding the item, of the piece of the file
				// the extent is part of, which the tiling of an image needs and
				// reading metadata does not.
				cursor.skip(indexSize)
			}

			extentOffset := cursor.long(offsetSize)
			extentLength := cursor.long(lengthSize)

			// The metadata of an AVIF file is held by a single piece, and the
			// extents of the other items are of no use to tinier.
			if extent == 0 {
				places[identifier] = place{
					offset: baseOffset + extentOffset,
					length: extentLength,
				}
			}
		}
	}
	if cursor.err != nil {
		return nil, truncatedErr(cursor.err)
	}

	return places, nil
}

// itemSegment returns the segment of a JPEG file holding the metadata of an item
// of an AVIF file, and a segment of no marker when the item holds metadata a
// JPEG file has no equivalent of.
func itemSegment(reader io.ReaderAt, definition item, where place) (segment, error) {
	if where.length <= 0 || where.length > maxMetadataBoxLength {
		return segment{}, fmt.Errorf("%w: the %s item of the file is %d bytes long",
			ErrInvalidImage, definition.name, where.length)
	}

	contents := make([]byte, where.length)
	if _, err := reader.ReadAt(contents, where.offset); err != nil {
		return segment{}, truncatedErr(err)
	}

	switch definition.name {
	case exifItemName:
		// The four bytes which start the item tell how far from them, and from
		// the end of the four, the EXIF structure starts, which writers make
		// zero.
		cursor := &cursor{contents: contents}
		structureOffset := versionAndFlagsSize + int(cursor.uint(versionAndFlagsSize))
		if cursor.err != nil || structureOffset >= len(contents) {
			return segment{}, truncatedErr(cursor.err)
		}

		return segment{
			marker:  app1Marker,
			payload: withIdentifier(contents[structureOffset:], exifIdentifier),
		}, nil

	case xmpItemName:
		// The item of an XMP packet is the packet itself, with no offset in
		// front of it, which the item of an EXIF structure has.
		return segment{
			marker:  app1Marker,
			payload: withIdentifier(contents, xmpIdentifier),
		}, nil
	}

	return segment{}, nil
}

// itemColorProfile returns the profile of the colors held by the box of the
// properties of the items of an AVIF file, and nil when the colors of the file
// are described by parameters instead of a profile, which is the case of most
// AVIF files.
func itemColorProfile(contents []byte) []byte {
	if len(contents) == 0 {
		return nil
	}

	propertyBoxes, err := containedBoxes(contents)
	if err != nil {
		return nil
	}

	for _, propertyBox := range propertyBoxes {
		if propertyBox.name != propertyContainerBoxName {
			continue
		}

		properties, err := containedBoxes(propertyBox.contents)
		if err != nil {
			return nil
		}

		for _, property := range properties {
			if profile := colorProfile(property.name, property.contents); profile != nil {
				return profile
			}
		}
	}

	return nil
}

// colorProfile returns the profile of the colors of a property of an item, and
// nil when the property describes the colors by parameters, which is what the
// primaries and the transfer characteristics of the colors do.
func colorProfile(name string, contents []byte) []byte {
	if name != colorInfoBoxName || len(contents) < boxNameSize {
		return nil
	}

	switch string(contents[:boxNameSize]) {
	case colorParametersName, colorLimitedRangeName:
		return nil
	default:
		// What is left of a box of color information which describes the colors
		// by parameters is an ICC profile, whatever the four characters naming it
		// are, iccColorProfileName and restrictedICCProfileName being the two
		// which hold one for sure.
		return contents[boxNameSize:]
	}
}

// box is a box of a base media file: its name and its contents.
type box struct {
	name     string
	contents []byte
}

// boxContents returns the contents of the first box of the given name of a list
// of boxes, and nil when there is none.
func boxContents(boxes []box, name string) []byte {
	for _, b := range boxes {
		if b.name == name {
			return b.contents
		}
	}

	return nil
}

// topLevelBoxContents returns the contents of the first box of the given name of
// a file, and tells if the file holds one. The boxes of the pixels of the file
// are walked over, and never read.
func topLevelBoxContents(reader io.ReaderAt, fileSize int64,
	name string) (contents []byte, found bool, err error) {
	for position := int64(0); position < fileSize; {
		h, err := readTopLevelBox(reader, position, fileSize)
		if err != nil {
			return nil, false, err
		}
		if h.name != name {
			position = h.next
			continue
		}

		if h.length > maxMetadataBoxLength {
			return nil, false, fmt.Errorf(
				"%w: the %s box of the file holds %d bytes",
				ErrInvalidImage, name, h.length)
		}

		contents = make([]byte, h.length)
		if _, err := reader.ReadAt(contents, h.offset); err != nil {
			return nil, false, truncatedErr(err)
		}

		return contents, true, nil
	}

	return nil, false, nil
}

// header is the header of a box of a base media file: what precedes its
// contents.
type header struct {
	name   string
	offset int64 // offset of the contents of the box.
	length int64 // length of the contents of the box.
	next   int64 // offset of the header of the box which follows.
}

// readTopLevelBox reads the header of the box of a file starting at the given
// position, and returns it. The length of a box is held by four bytes, unless it
// is one, which means the length is held by eight bytes which follow the name,
// or zero, which means the box reaches the end of the file.
func readTopLevelBox(reader io.ReaderAt, position, fileSize int64) (h header, err error) {
	shortHeader := make([]byte, boxLengthSize+boxNameSize)
	if _, err := reader.ReadAt(shortHeader, position); err != nil {
		return header{}, truncatedErr(err)
	}

	h.name = string(shortHeader[boxLengthSize:])
	length, ok := signed(uint64(binary.BigEndian.Uint32(shortHeader[:boxLengthSize])))
	if !ok {
		return header{}, truncatedErr(nil)
	}
	headerSize := int64(boxLengthSize + boxNameSize)

	switch length {
	case 1:
		// The real length of the box is held by the eight bytes which directly
		// follow the name of the box.
		longLength := make([]byte, longBoxLengthSize)
		if _, err := reader.ReadAt(longLength, position+headerSize); err != nil {
			return header{}, truncatedErr(err)
		}
		length, ok = signed(binary.BigEndian.Uint64(longLength))
		if !ok {
			return header{}, truncatedErr(nil)
		}
		headerSize += longBoxLengthSize

	case 0:
		length = fileSize - position
	}

	if length < headerSize {
		return header{}, fmt.Errorf(
			"%w: the %s box of the file is shorter than its header", ErrInvalidImage, h.name)
	}

	// The length held by the header counts the header itself, which the caller
	// reading the contents of the box has no use of.
	h.offset = position + headerSize
	h.length = length - headerSize
	h.next = position + length

	if h.next > fileSize {
		return header{}, fmt.Errorf(
			"%w: the %s box of the file ends past the file", ErrInvalidImage, h.name)
	}

	return h, nil
}

// containedBoxes returns the boxes held by the contents of a box, in the order
// the contents hold them. A box of length zero reaches the end of the contents
// which hold it, and a box whose length is one holds its real length in the
// eight bytes which follow its name.
func containedBoxes(contents []byte) ([]box, error) {
	var boxes []box

	for position := 0; position < len(contents); {
		if len(contents)-position < boxLengthSize+boxNameSize {
			return nil, truncatedErr(nil)
		}

		headerSize := boxLengthSize + boxNameSize
		length, ok := signed(uint64(binary.BigEndian.Uint32(contents[position:])))
		if !ok {
			return nil, truncatedErr(nil)
		}

		switch length {
		case 1:
			if len(contents)-position < headerSize+longBoxLengthSize {
				return nil, truncatedErr(nil)
			}
			start := position + headerSize
			length, ok = signed(binary.BigEndian.Uint64(contents[start : start+longBoxLengthSize]))
			if !ok {
				return nil, truncatedErr(nil)
			}
			headerSize += longBoxLengthSize

		case 0:
			length = int64(len(contents) - position)
		}

		if length < int64(headerSize) {
			return nil, truncatedErr(nil)
		}
		end := int64(position) + length
		if end > int64(len(contents)) {
			return nil, fmt.Errorf(
				"%w: the %s box of the file is truncated",
				ErrInvalidImage, contents[position+boxLengthSize:position+headerSize])
		}

		// The name of a box always directly follows the four bytes of its length,
		// even when the real length of a box of length one is held by the eight
		// bytes which follow the name.
		nameStart := position + boxLengthSize
		boxes = append(boxes, box{
			name:     string(contents[nameStart : nameStart+boxNameSize]),
			contents: contents[position+headerSize : end],
		})
		position = int(end)
	}

	return boxes, nil
}

// signed turns the length or the offset a box of a file holds into a signed
// number, and tells if it is one. A file tinier can read never reaches the size
// an unsigned number of eight bytes describes, and a box announcing it holds a
// corrupted one.
func signed(value uint64) (int64, bool) {
	if value > math.MaxInt64 {
		return 0, false
	}

	return int64(value), true
}

// cursor reads, one after the other, the integers of the contents of a box. The
// sizes of the fields of a base media file are packed four bits by four bits,
// and are therefore known only while reading them. Any read past the contents,
// which is what a file holding fewer bytes than its boxes announce is made of,
// fails the cursor.
type cursor struct {
	contents []byte
	position int
	err      error
}

// uint returns the integer of the given size, four bytes at most, the cursor is
// on. A size of zero means the field is not there at all, which the format makes
// use of.
func (c *cursor) uint(size int) uint32 {
	if c.err != nil {
		return 0
	}
	if size < 0 || size > longFieldSize || c.position+size > len(c.contents) {
		c.err = truncatedErr(nil)
		return 0
	}
	if size == 0 {
		return 0
	}

	// The integer of the field is right aligned in a number of four bytes, so
	// that the numbers of one, two and four bytes are read the same way.
	var padded [longFieldSize]byte
	copy(padded[longFieldSize-size:], c.contents[c.position:c.position+size])
	value := binary.BigEndian.Uint32(padded[:])
	c.position += size

	return value
}

// long returns the integer of the given size, eight bytes at most, the cursor is
// on, as a signed number: an offset or a length of a file is never negative, and
// a field of eight bytes above the largest signed number describes a file tinier
// cannot read, which fails the cursor.
func (c *cursor) long(size int) int64 {
	if c.err != nil {
		return 0
	}
	if size < 0 || size > maxFieldSize || c.position+size > len(c.contents) {
		c.err = truncatedErr(nil)
		return 0
	}
	if size == 0 {
		return 0
	}

	// The integer of the field is right aligned in a number of eight bytes, so
	// that the numbers of one, two, four and eight bytes are read the same way.
	var padded [maxFieldSize]byte
	copy(padded[maxFieldSize-size:], c.contents[c.position:c.position+size])
	value, ok := signed(binary.BigEndian.Uint64(padded[:]))
	if !ok {
		c.err = truncatedErr(nil)
		return 0
	}
	c.position += size

	return value
}

// name returns the four characters the cursor is on, which name a box or an item
// of a base media file.
func (c *cursor) name() string {
	if c.err != nil {
		return ""
	}
	if c.position+boxNameSize > len(c.contents) {
		c.err = truncatedErr(nil)
		return ""
	}

	name := string(c.contents[c.position : c.position+boxNameSize])
	c.position += boxNameSize

	return name
}

// skip moves the cursor past the given number of bytes, which are of no use to
// tinier.
func (c *cursor) skip(size int) {
	if c.err != nil {
		return
	}
	if size < 0 || c.position+size > len(c.contents) {
		c.err = truncatedErr(nil)
		return
	}

	c.position += size
}

// truncatedErr returns the error telling an image file ends inside a box,
// wrapping the given error when there is one.
func truncatedErr(err error) error {
	if err == nil {
		return fmt.Errorf("%w: the file ends inside a box", ErrInvalidImage)
	}

	return fmt.Errorf("%w: the file ends inside a box: %w", ErrInvalidImage, err)
}
