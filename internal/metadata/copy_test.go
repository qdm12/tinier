package metadata

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopy copies the metadata of a JPEG file into another one, and updates the
// tags which describe the pixels of the destination file.
func TestCopy(t *testing.T) {
	srcPath := writeFixture(t, "source.jpg", jpeg(
		[]segment{
			{marker: app13Marker, payload: iptcSegment()},
			{marker: 0xE0, payload: []byte("JFIF source\x00")},
			{marker: app2Marker, payload: iccSegment()},
			{marker: app1Marker, payload: exifSegment()},
			{marker: commentMarker, payload: []byte("a photographer")},
			{marker: app1Marker, payload: xmpSegment()},
			{marker: 0xDB, payload: []byte("a quantization table")},
			startOfFrameSegment(100, 200),
		},
		startOfScanMarker,
		[]byte("source pixels"),
	))
	dstPath := writeFixture(t, "destination.jpg", jpeg(
		[]segment{
			{marker: 0xE0, payload: []byte("JFIF destination\x00")},
			{marker: commentMarker, payload: []byte("Lavc60.16.100")},
			{marker: 0xDB, payload: []byte("another quantization table")},
			startOfFrameSegment(200, 100),
		},
		startOfScanMarker,
		[]byte("destination pixels"),
	))
	require.NoError(t, os.Chmod(dstPath, 0640))

	require.NoError(t, Copy(dstPath, srcPath))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)

	segments, endMarker, err := readHeader(newReader(data))
	require.NoError(t, err)
	require.Equal(t, startOfScanMarker, endMarker)

	kinds := make([]kind, len(segments))
	for i, s := range segments {
		kinds[i] = s.kind()
	}

	assert.Equal(t, []kind{
		kindExif, kindXMP, kindICCProfile, kindIPTC, kindComment,
		kindOther, kindOther, kindOther,
	}, kinds, "the metadata comes first, in the copy order, and replaces the segments of the same kind")

	assert.Equal(t, []byte("a photographer"), segments[4].payload,
		"the comment of the input file replaces the one of the encoder")
	assert.Equal(t, []byte("JFIF destination\x00"), segments[5].payload,
		"the segments which hold no metadata are kept")
	assert.NotContains(t, string(data), "JFIF source", "the segments of the input file are not copied")

	assert.Contains(t, string(data), "destination pixels")
	assert.NotContains(t, string(data), "source pixels")
	assert.True(t, bytes.HasSuffix(data, []byte{markerPrefix, endOfImageMarker}), "the file ends with its image")

	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm(), "the permissions of the file are kept")

	// The rotation of the input file is in the pixels of the destination file,
	// and its dimensions are the ones of the destination file.
	structure, err := newTiff(segments[0].payload[len(exifIdentifier):])
	require.NoError(t, err)

	assert.Equal(t, orientationNormal, valueOf(t, structure, structure.firstIFDOffset(), orientationTag))
	assert.Equal(t, uint32(200), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag))
	assert.Equal(t, uint32(100), valueOf(t, structure, structure.firstIFDOffset(), imageLengthTag))

	subIFD := ifd1Of(t, structure)
	assert.Equal(t, uint32(200), valueOf(t, structure, subIFD, pixelXDimensionTag))
	assert.Equal(t, uint32(100), valueOf(t, structure, subIFD, pixelYDimensionTag))

	assert.Equal(t, makerNoteBytes(), makerNoteOf(segments[0].payload),
		"the maker note is copied as it is")
}

// TestCopyNoMetadata tells that the destination file is left as it is when the
// source file holds no metadata.
func TestCopyNoMetadata(t *testing.T) {
	srcPath := writeFixture(t, "source.jpg", jpeg(
		[]segment{startOfFrameSegment(100, 200)},
		startOfScanMarker,
		[]byte("source pixels"),
	))

	dstData := jpeg(
		[]segment{
			{marker: commentMarker, payload: []byte("Lavc60.16.100")},
			startOfFrameSegment(200, 100),
		},
		startOfScanMarker,
		[]byte("destination pixels"),
	)
	dstPath := writeFixture(t, "destination.jpg", dstData)

	require.NoError(t, Copy(dstPath, srcPath))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, dstData, data)
}

// TestCopyUnsupportedFormat tells that an error matching ErrUnsupportedFormat is
// returned, and the destination file left as it is, when one of the files is not
// in a format tinier reads the metadata of.
func TestCopyUnsupportedFormat(t *testing.T) {
	jpegPath := writeFixture(t, "image.jpg", jpeg(
		[]segment{
			{marker: app1Marker, payload: exifSegment()},
			startOfFrameSegment(100, 200),
		},
		startOfScanMarker,
		[]byte("pixels"),
	))
	// The AVIF and HEIC files, which tinier converts, are made of ISO base media
	// file format boxes, which hold no JPEG segment nor PNG chunk.
	heicData := []byte("\x00\x00\x00 ftypheic")
	heicPath := writeFixture(t, "image.heic", heicData)

	t.Run("the source is not a supported file", func(t *testing.T) {
		dstData := jpeg([]segment{startOfFrameSegment(100, 200)}, startOfScanMarker, []byte("pixels"))
		dstPath := writeFixture(t, "destination.jpg", dstData)

		err := Copy(dstPath, heicPath)
		assert.ErrorIs(t, err, ErrUnsupportedFormat)

		after, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, dstData, after, "the destination file is left as it is")
	})

	t.Run("the destination is not a jpeg file", func(t *testing.T) {
		dstPath := writeFixture(t, "destination.heic", heicData)

		err := Copy(dstPath, jpegPath)
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})
}

// TestCopyTruncatedDestination tells that an error matching ErrInvalidImage is
// returned when the header of the destination file cannot be parsed.
func TestCopyTruncatedDestination(t *testing.T) {
	srcPath := writeFixture(t, "source.jpg", jpeg(
		[]segment{
			{marker: app1Marker, payload: exifSegment()},
			startOfFrameSegment(100, 200),
		},
		startOfScanMarker,
		[]byte("pixels"),
	))
	dstPath := writeFixture(t, "destination.jpg", truncatedJPEG(0x00, 0xFF))

	err := Copy(dstPath, srcPath)

	assert.ErrorIs(t, err, ErrInvalidImage)
}

// TestCopyDestinationWithoutScan tells that the metadata is copied into a
// destination file holding no scan, the header of which ends with the end of
// image marker.
func TestCopyDestinationWithoutScan(t *testing.T) {
	srcPath := writeFixture(t, "source.jpg", jpeg(
		[]segment{
			{marker: app1Marker, payload: exifSegment()},
			startOfFrameSegment(100, 200),
		},
		startOfScanMarker,
		[]byte("pixels"),
	))
	dstPath := writeFixture(t, "destination.jpg", jpeg(
		[]segment{startOfFrameSegment(200, 100)},
		endOfImageMarker,
		nil,
	))

	require.NoError(t, Copy(dstPath, srcPath))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)

	segments, endMarker, err := readHeader(newReader(data))
	require.NoError(t, err)
	assert.Equal(t, endOfImageMarker, endMarker)

	kinds := make([]kind, 0, len(segments))
	for _, s := range segments {
		kinds = append(kinds, s.kind())
	}
	assert.Contains(t, kinds, kindExif)
}

// TestCopyMissingFiles tells that an error is returned when one of the files
// does not exist.
func TestCopyMissingFiles(t *testing.T) {
	jpegPath := writeFixture(t, "image.jpg", jpeg(
		[]segment{
			{marker: app1Marker, payload: exifSegment()},
			startOfFrameSegment(100, 200),
		},
		startOfScanMarker,
		[]byte("pixels"),
	))
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.jpg")

	metadata, err := readMetadata(missingPath)
	assert.Error(t, err)
	assert.Empty(t, metadata)

	assert.Error(t, Copy(missingPath, jpegPath), "the destination file does not exist")
}

// TestSortSegments tells that the segments are sorted by kind, that the segments
// of the same kind keep their order, and that the given segments are not
// modified.
func TestSortSegments(t *testing.T) {
	segments := []segment{
		{marker: commentMarker, payload: []byte("comment first in file")},
		{marker: app13Marker, payload: iptcSegment()},
		{marker: app1Marker, payload: xmpSegment()},
		{marker: 0xDB, payload: []byte("a quantization table")},
		{marker: app1Marker, payload: exifSegment()},
		{marker: commentMarker, payload: []byte("comment second in file")},
	}

	sorted := sortSegments(segments)

	require.Len(t, sorted, 5, "the segments which hold no metadata are left out")
	assert.Equal(t, kindExif, sorted[0].kind())
	assert.Equal(t, kindXMP, sorted[1].kind())
	assert.Equal(t, kindIPTC, sorted[2].kind())
	assert.Equal(t, kindComment, sorted[3].kind())
	assert.Equal(t, []byte("comment first in file"), sorted[3].payload,
		"the segments of the same kind keep the order they have in the file")
	assert.Equal(t, []byte("comment second in file"), sorted[4].payload)
	assert.Len(t, segments, 6)
}

// TestSegmentKind tells the kind of metadata each segment carries.
func TestSegmentKind(t *testing.T) {
	tests := []struct {
		name     string
		segment  segment
		expected kind
	}{
		{"exif", segment{marker: app1Marker, payload: exifSegment()}, kindExif},
		{"xmp", segment{marker: app1Marker, payload: xmpSegment()}, kindXMP},
		{"extended xmp", segment{
			marker:  app1Marker,
			payload: append([]byte(xmpExtendedIdentifier), "xmp"...),
		}, kindXMP},
		{"icc profile", segment{marker: app2Marker, payload: iccSegment()}, kindICCProfile},
		{"iptc written by photoshop", segment{marker: app13Marker, payload: iptcSegment()}, kindIPTC},
		{"raw iptc", segment{
			marker:  app13Marker,
			payload: append([]byte(iptcRawIdentifier), "iptc"...),
		}, kindIPTC},
		{"comment", segment{marker: commentMarker, payload: []byte("a comment")}, kindComment},
		{"jiff", segment{marker: 0xE0, payload: []byte("JFIF\x00")}, kindOther},
		{"unknown app1", segment{marker: app1Marker, payload: []byte("something else")}, kindOther},
		{"icc profile without identifier", segment{marker: app2Marker, payload: []byte("a profile")}, kindOther},
		{"quantization table", segment{marker: 0xDB, payload: []byte("a table")}, kindOther},
		{"start of frame", segment{marker: 0xC0, payload: []byte{8, 0, 1, 0, 2}}, kindOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.segment.kind())
		})
	}
}

// TestFrameSize tells the dimensions a start of frame segment holds.
func TestFrameSize(t *testing.T) {
	frame := startOfFrameSegment(3072, 1728)

	width, height, ok := frame.frameSize()

	require.True(t, ok)
	assert.Equal(t, uint32(3072), width)
	assert.Equal(t, uint32(1728), height)

	t.Run("not a start of frame segment", func(t *testing.T) {
		table := segment{marker: 0xDB, payload: []byte{8, 0, 1, 0, 2}}

		_, _, ok := table.frameSize()

		assert.False(t, ok)
	})

	t.Run("too short to hold dimensions", func(t *testing.T) {
		truncated := segment{marker: 0xC0, payload: []byte{8, 0, 1}}

		_, _, ok := truncated.frameSize()

		assert.False(t, ok)
	})
}

// TestEndsHeader tells which markers end the header of a JPEG file.
func TestEndsHeader(t *testing.T) {
	for _, marker := range []uint8{restartMarkerFirst, restartMarkerLast, startOfScanMarker, endOfImageMarker} {
		assert.True(t, endsHeader(marker), "the marker %#X ends the header", marker)
	}

	for _, marker := range []uint8{startOfImageMarker, app1Marker, app2Marker, commentMarker, 0xC0} {
		assert.False(t, endsHeader(marker), "the marker %#X does not end the header", marker)
	}
}

// TestReadHeader tells that the header of a JPEG file is read, and that the
// reader is left after the marker which ends it.
func TestReadHeader(t *testing.T) {
	data := jpeg(
		[]segment{
			{marker: 0xE0, payload: []byte("JFIF\x00")},
			{marker: commentMarker, payload: []byte("a comment")},
		},
		startOfScanMarker,
		[]byte("the pixels"),
	)

	reader := newReader(data)

	segments, endMarker, err := readHeader(reader)
	require.NoError(t, err)

	assert.Equal(t, startOfScanMarker, endMarker)
	require.Len(t, segments, 2)
	assert.Equal(t, []byte("JFIF\x00"), segments[0].payload)
	assert.Equal(t, []byte("a comment"), segments[1].payload)

	rest, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, append([]byte("the pixels"), endOfImageBytes()...), rest)
}

// TestReadHeaderErrors tells the errors returned for the files which are not
// JPEG files, or which are corrupted ones.
func TestReadHeaderErrors(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected error
	}{
		{"empty", nil, ErrUnsupportedFormat},
		{"png", []byte("\x89PNG\r\n\x1a\n"), ErrUnsupportedFormat},
		{"no marker at all", []byte{'H', 'e', 'l', 'l', 'o'}, ErrUnsupportedFormat},
		{"truncated length", truncatedJPEG(0x00), ErrInvalidImage},
		{"length below its own size", truncatedJPEG(0x00, 0x01, 0x00), ErrInvalidImage},
		{"truncated payload", truncatedJPEG(0x00, 0x10, 0x00), ErrInvalidImage},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := readHeader(newReader(test.data))

			assert.ErrorIs(t, err, test.expected)
		})
	}
}

// TestReconcileMetadata tells that the tags which describe the pixels are
// updated, following the byte order of the EXIF payload, and that a segment
// holding no valid EXIF payload is dropped instead of describing the pixels of
// another file.
func TestReconcileMetadata(t *testing.T) {
	t.Run("the dimensions of the destination are unknown", func(t *testing.T) {
		metadata := []segment{{marker: app1Marker, payload: exifSegment()}}

		reconciled := reconcileMetadata(metadata, 0, 0)

		require.Len(t, reconciled, 1)
		structure, err := newTiff(reconciled[0].payload[len(exifIdentifier):])
		require.NoError(t, err)
		assert.Equal(t, orientationNormal, valueOf(t, structure, structure.firstIFDOffset(), orientationTag))
		assert.Equal(t, uint32(100), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag),
			"the dimensions are left as they are when they are unknown")
	})

	t.Run("the exif payload is invalid", func(t *testing.T) {
		payload := append([]byte(exifIdentifier), "not a tiff structure"...)
		metadata := []segment{
			{marker: app1Marker, payload: payload},
			{marker: commentMarker, payload: []byte("a comment")},
		}

		reconciled := reconcileMetadata(metadata, 200, 100)

		require.Len(t, reconciled, 1, "the exif which cannot describe the pixels is dropped")
		assert.Equal(t, kindComment, reconciled[0].kind())
	})

	t.Run("the exif payload is big endian", func(t *testing.T) {
		metadata := []segment{{
			marker:  app1Marker,
			payload: append([]byte(exifIdentifier), bigEndianEXIFBytes()...),
		}}

		reconciled := reconcileMetadata(metadata, 1280, 720)

		require.Len(t, reconciled, 1)
		structure, err := newTiff(reconciled[0].payload[len(exifIdentifier):])
		require.NoError(t, err)
		assert.Equal(t, orientationNormal, valueOf(t, structure, structure.firstIFDOffset(), orientationTag))
		assert.Equal(t, uint32(1280), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag),
			"the dimensions are written following the byte order of the payload")
	})
}

// TestFixtures checks that the fixtures hold what they declare, so the tests
// which read them mean what they say.
func TestFixtures(t *testing.T) {
	structure, err := newTiff(exifBytes())
	require.NoError(t, err)

	assert.Equal(t, ifd0Offset, structure.firstIFDOffset())
	assert.Equal(t, uint32(6), valueOf(t, structure, structure.firstIFDOffset(), orientationTag))
	assert.Equal(t, uint32(100), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag))
	assert.Equal(t, uint32(200), valueOf(t, structure, structure.firstIFDOffset(), imageLengthTag))

	subIFD := ifd1Of(t, structure)
	assert.Equal(t, ifd1Offset, subIFD)
	assert.Equal(t, uint32(100), valueOf(t, structure, subIFD, pixelXDimensionTag))
	assert.Equal(t, uint32(200), valueOf(t, structure, subIFD, pixelYDimensionTag))
	assert.Equal(t, uint32(makerNoteOffset), offsetOf(t, structure, subIFD, makerNoteTag))

	assert.Len(t, exifBytes(), exifStructureLen)
	assert.Equal(t, makerNoteBytes(), exifBytes()[makerNoteOffset:makerNoteOffset+makerNoteSize])
}

// valueOf returns the value of the entry of the given tag, in the image file
// directory at the given offset.
func valueOf(t *testing.T, structure *tiff, ifdOffset int, tag uint16) uint32 {
	t.Helper()

	e, ok := structure.entry(ifdOffset, tag)
	require.Truef(t, ok, "the tag %#X is absent", tag)

	value, ok := structure.uint(e)
	require.Truef(t, ok, "the tag %#X holds no single integer", tag)

	return value
}

// offsetOf returns the offset the entry of the given tag points to, in the image
// file directory at the given offset.
func offsetOf(t *testing.T, structure *tiff, ifdOffset int, tag uint16) uint32 {
	t.Helper()

	e, ok := structure.entry(ifdOffset, tag)
	require.Truef(t, ok, "the tag %#X is absent", tag)
	require.Greaterf(t, e.count, uint32(1), "the tag %#X does not point to a value", tag)

	return binary.LittleEndian.Uint32(structure.data[e.valueOffset:])
}

// ifd1Of returns the offset of the EXIF sub directory of a TIFF structure.
func ifd1Of(t *testing.T, structure *tiff) int {
	t.Helper()

	offset, ok := structure.subIFDOffset(structure.firstIFDOffset(), exifSubIFDPointerTag)
	require.True(t, ok, "the exif sub directory is absent")

	return offset
}

// truncatedJPEG returns the bytes of a JPEG file whose first segment is an EXIF
// segment announced by the given length bytes, which cut it short.
func truncatedJPEG(lengthBytes ...byte) []byte {
	data := make([]byte, 0, 4+len(lengthBytes))
	data = append(data, markerPrefix, startOfImageMarker, markerPrefix, app1Marker)
	return append(data, lengthBytes...)
}

// maxPayloadLength is the largest payload a JPEG segment can hold, since the
// length it announces counts its own two bytes.
const maxPayloadLength = 0xFFFF - segmentSize

// TestSortSegmentsDuplicates tells that only the first segment of each kind is
// kept, except the ICC profiles, which a file splits into as many segments as
// it needs, and the comments, which are not duplicates of one another.
func TestSortSegmentsDuplicates(t *testing.T) {
	segments := []segment{
		{marker: app1Marker, payload: exifSegment()},
		{marker: app1Marker, payload: exifSegment()},
		{marker: app1Marker, payload: xmpSegment()},
		{marker: app1Marker, payload: xmpSegment()},
		{marker: app2Marker, payload: iccSegment()},
		{marker: app2Marker, payload: iccSegment()},
		{marker: commentMarker, payload: []byte("first comment")},
		{marker: commentMarker, payload: []byte("second comment")},
	}

	sorted := sortSegments(segments)

	require.Len(t, sorted, 6)
	assert.Equal(t, kindExif, sorted[0].kind())
	assert.Equal(t, kindXMP, sorted[1].kind())
	assert.Equal(t, kindICCProfile, sorted[2].kind())
	assert.Equal(t, kindICCProfile, sorted[3].kind())
	assert.Equal(t, kindComment, sorted[4].kind())
	assert.Equal(t, kindComment, sorted[5].kind())
}

// TestReadHeaderTrickySegments tells that a segment of the largest payload a
// JPEG file can announce is read, that the marker bytes held by a payload do
// not cut the header short, and that a segment of no payload is read.
func TestReadHeaderTrickySegments(t *testing.T) {
	tests := []struct {
		name    string
		segment segment
	}{
		{
			name:    "the largest payload a segment can hold",
			segment: segment{marker: app1Marker, payload: bytes.Repeat([]byte("x"), maxPayloadLength)},
		},
		{
			name:    "the marker bytes held by a payload",
			segment: segment{marker: app1Marker, payload: []byte("a \xff\xd9 and a \xff\xda inside")},
		},
		{
			name:    "no payload at all",
			segment: segment{marker: commentMarker, payload: []byte{}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := jpeg([]segment{test.segment, startOfFrameSegment(100, 200)}, startOfScanMarker, []byte("pixels"))

			segments, endMarker, err := readHeader(newReader(data))

			require.NoError(t, err)
			assert.Equal(t, startOfScanMarker, endMarker)
			require.Len(t, segments, 2)
			assert.Equal(t, test.segment.payload, segments[0].payload,
				"the payload is read untouched")
		})
	}
}

// TestCopyInvalidExif tells that the EXIF which cannot describe the pixels of
// the destination file is dropped, and the other metadata copied.
func TestCopyInvalidExif(t *testing.T) {
	srcPath := writeFixture(t, "source.jpg", jpeg([]segment{
		{marker: app1Marker, payload: append([]byte(exifIdentifier), "not a tiff structure"...)},
		{marker: commentMarker, payload: []byte("a comment")},
		startOfFrameSegment(100, 200),
	}, startOfScanMarker, []byte("pixels")))
	dstPath := writeFixture(t, "destination.jpg", jpeg([]segment{
		{marker: commentMarker, payload: []byte("Lavc")},
		startOfFrameSegment(200, 100),
	}, startOfScanMarker, []byte("destination pixels")))

	require.NoError(t, Copy(dstPath, srcPath))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.True(t, bytes.Contains(data, []byte("destination pixels")))

	segments, _, err := readHeader(newReader(data))
	require.NoError(t, err)

	kinds := make([]kind, 0, len(segments))
	for _, s := range segments {
		kinds = append(kinds, s.kind())
	}
	assert.NotContains(t, kinds, kindExif)
	assert.Contains(t, kinds, kindComment)
}

// TestTransplantedLeavesNoTemporaryFile tells that no temporary file is left
// behind when the destination file cannot be replaced.
func TestTransplantedLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	dstPath := filepath.Join(dir, "image.jpg")
	require.NoError(t, os.WriteFile(dstPath,
		jpeg([]segment{startOfFrameSegment(200, 100)}, startOfScanMarker, []byte("pixels")), 0600))
	metadata := []segment{{marker: commentMarker, payload: []byte("a comment")}}

	_, err := writeTransplanted(dstPath, metadata, nil, startOfScanMarker, failingReader{})

	require.Error(t, err)
	names, err := filepath.Glob(filepath.Join(dir, ".tinier-metadata-*"))
	require.NoError(t, err)
	assert.Empty(t, names, "the temporary file is removed when it cannot replace the destination file")
}

// failingReader is a reader which always fails, as the reader of the pixels of
// a file being read from a disk which went away.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("cannot read the pixels")
}

// writeFixture writes data to a file of a temporary directory and returns its
// path.
func writeFixture(t *testing.T, name string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, data, 0600))

	return path
}

// newReader returns a buffered reader of the given data.
func newReader(data []byte) *bufio.Reader {
	return bufio.NewReader(bytes.NewReader(data))
}

// jpeg returns the bytes of a JPEG file made of the given segments, followed by
// the marker which ends the header, the given pixels, and the end of image
// marker.
func jpeg(segments []segment, endMarker uint8, pixels []byte) []byte {
	data := []byte{markerPrefix, startOfImageMarker}

	for _, s := range segments {
		data = append(data, s.appended()...)
	}

	data = append(data, markerPrefix, endMarker)
	data = append(data, pixels...)

	return append(data, endOfImageBytes()...)
}

// endOfImageBytes returns the bytes of the end of image marker.
func endOfImageBytes() []byte {
	return []byte{markerPrefix, endOfImageMarker}
}

// startOfFrameSegment returns the segment which holds the dimensions of an
// image.
func startOfFrameSegment(width, height uint16) segment {
	payload := make([]byte, 5)
	payload[0] = 8 // sample precision, bits per sample
	binary.BigEndian.PutUint16(payload[1:3], height)
	binary.BigEndian.PutUint16(payload[3:5], width)

	return segment{marker: 0xC0, payload: payload}
}

// xmpSegment returns the payload of the segment holding the XMP metadata of the
// fixtures.
func xmpSegment() []byte {
	return append([]byte(xmpIdentifier), "<x:xmpmeta>made by tinier</x:xmpmeta>"...)
}

// iccSegment returns the payload of the segment holding the color profile of
// the fixtures.
func iccSegment() []byte {
	return append([]byte(iccProfileIdentifier), "a color profile"...)
}

// iptcSegment returns the payload of the segment holding the IPTC metadata of
// the fixtures, as Photoshop writes it.
func iptcSegment() []byte {
	return append([]byte(photoshopIdentifier), "8BIM\x04\x04\x00\x00 IPTC data"...)
}

// makerNoteTag is the tag of the maker note of the fixtures, an application
// defined tag holding information no standard describes.
const makerNoteTag uint16 = 0x9000

// Offsets and sizes of the EXIF fixture, relative to the start of its TIFF
// structure, which the payload of the EXIF segment holds after its identifier.
// The size of an image file directory is 2 bytes counting its entries, 12 bytes
// per entry, and 4 bytes holding the offset of the next directory.
const (
	ifd0Offset       = 8
	ifd0EntryCount   = 4
	ifd1Offset       = ifd0Offset + 2 + ifd0EntryCount*12 + 4
	ifd1EntryCount   = 3
	makerNoteOffset  = ifd1Offset + 2 + ifd1EntryCount*12 + 4
	makerNoteSize    = 15
	exifStructureLen = makerNoteOffset + makerNoteSize
)

// exifSegment returns the payload of the EXIF segment of the fixtures. Its
// orientation is the one of an image stored rotated by 90 degrees clockwise,
// sized 100 by 200.
func exifSegment() []byte {
	return append([]byte(exifIdentifier), exifBytes()...)
}

// exifBytes returns the bytes of the little endian TIFF structure of the EXIF
// fixture, made of a first image file directory, of an EXIF sub directory, and
// of a maker note stored out of the directories:
//
//	0:   the byte order 'I' 'I', the magic number, and the offset of IFD0
//	8:   IFD0, holding the dimensions, the orientation, and the sub directory
//	62:  the EXIF sub directory, holding the dimensions and the maker note
//	104: the maker note, 15 bytes
func exifBytes() []byte {
	structure := make([]byte, 0, exifStructureLen)
	structure = append(structure, 'I', 'I')
	structure = binary.LittleEndian.AppendUint16(structure, tiffMagic)
	structure = binary.LittleEndian.AppendUint32(structure, ifd0Offset)

	structure = appendIFD(structure, binary.LittleEndian, []entry{
		{tag: imageWidthTag, dataType: typeLong, count: 1, value: 100},
		{tag: imageLengthTag, dataType: typeLong, count: 1, value: 200},
		{tag: orientationTag, dataType: typeShort, count: 1, value: 6},
		{tag: exifSubIFDPointerTag, dataType: typeLong, count: 1, value: ifd1Offset},
	}, 0)

	structure = appendIFD(structure, binary.LittleEndian, []entry{
		{tag: pixelXDimensionTag, dataType: typeShort, count: 1, value: 100},
		{tag: pixelYDimensionTag, dataType: typeShort, count: 1, value: 200},
		{tag: makerNoteTag, dataType: typeUndefined, count: makerNoteSize, value: makerNoteOffset, outOfLine: true},
	}, 0)

	return append(structure, makerNoteBytes()...)
}

// makerNoteBytes returns the bytes of the maker note of the fixtures.
func makerNoteBytes() []byte {
	return []byte("MakerNoteData!\x00")
}

// makerNoteOf returns the maker note held by the payload of an EXIF segment.
func makerNoteOf(payload []byte) []byte {
	return payload[len(exifIdentifier)+makerNoteOffset : len(exifIdentifier)+makerNoteOffset+makerNoteSize]
}

// bigEndianEXIFBytes returns the bytes of a big endian TIFF structure holding an
// orientation of 90 degrees clockwise, sized 100 by 0.
func bigEndianEXIFBytes() []byte {
	structure := make([]byte, 0, 38)
	structure = append(structure, 'M', 'M')
	structure = binary.BigEndian.AppendUint16(structure, tiffMagic)
	structure = binary.BigEndian.AppendUint32(structure, ifd0Offset)

	return appendIFD(structure, binary.BigEndian, []entry{
		{tag: imageWidthTag, dataType: typeLong, count: 1, value: 100},
		{tag: orientationTag, dataType: typeShort, count: 1, value: 6},
	}, 0)
}

// appender appends the values of a TIFF structure to a slice of bytes, in the
// byte order of the structure.
type appender interface {
	AppendUint16(dst []byte, v uint16) []byte
	AppendUint32(dst []byte, v uint32) []byte
}

// entry is an entry of the image file directories of the fixtures. The value
// field holds the value of the entry when it fits in its 4 bytes, and the
// offset of its value otherwise.
type entry struct {
	tag       uint16
	dataType  uint16
	count     uint32
	value     uint32
	outOfLine bool
}

// appendIFD appends an image file directory holding the given entries, followed
// by the offset of the next directory, which is zero when there is none.
func appendIFD(structure []byte, order appender, entries []entry, nextIFDOffset uint32) []byte {
	// A fixture holds a handful of entries, which fits in the 2 bytes of the
	// count of an image file directory.
	structure = order.AppendUint16(structure, uint16(len(entries))) //nolint:gosec

	for _, e := range entries {
		structure = order.AppendUint16(structure, e.tag)
		structure = order.AppendUint16(structure, e.dataType)
		structure = order.AppendUint32(structure, e.count)
		structure = appendEntryValue(structure, order, e)
	}

	return order.AppendUint32(structure, nextIFDOffset)
}

// appendEntryValue appends the 4 bytes which hold the value of an entry, or the
// offset of its value when the value is stored out of the directory.
func appendEntryValue(structure []byte, order appender, e entry) []byte {
	switch {
	case e.outOfLine:
		return order.AppendUint32(structure, e.value)
	case e.dataType == typeShort:
		// The value of a short entry fits in the 2 bytes it takes.
		structure = order.AppendUint16(structure, uint16(e.value)) //nolint:gosec
		return append(structure, 0, 0)
	default:
		return order.AppendUint32(structure, e.value)
	}
}

// TestReadHeaderErrorKinds checks the kind of error a header which cannot be
// read returns. It matters because Copy leaves a file of an unsupported format
// in silence, when the metadata a broken file could have given has to be
// missed. Only the file which does not start like a JPEG file is of another
// format: past its first marker, it is a JPEG file, and a header that cannot be
// read makes it an invalid one.
func TestReadHeaderErrorKinds(t *testing.T) {
	header := []byte{markerPrefix, startOfImageMarker}

	tests := map[string]struct {
		data          []byte
		wantedError   error
		unwantedError error
	}{
		"of another format": {
			data:        []byte{'P', 'N', 'G', '!'},
			wantedError: ErrUnsupportedFormat,
		},
		"segment without its prefix byte": {
			data:          append(header, app1Marker, 0, 2),
			wantedError:   ErrInvalidImage,
			unwantedError: ErrUnsupportedFormat,
		},
		"header cut short": {
			data:          append(header, markerPrefix),
			wantedError:   ErrInvalidImage,
			unwantedError: ErrUnsupportedFormat,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			_, _, err := readHeader(bufio.NewReader(bytes.NewReader(test.data)))

			require.ErrorIs(t, err, test.wantedError)
			if test.unwantedError != nil {
				assert.NotErrorIs(t, err, test.unwantedError)
			}
		})
	}
}
