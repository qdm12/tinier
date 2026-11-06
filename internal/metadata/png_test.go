package metadata

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetadataChunk tells which chunks of a PNG file hold metadata a JPEG file
// can express, and how that metadata is written in the segment of a JPEG file.
func TestMetadataChunk(t *testing.T) {
	profile := []byte("icc profile of the png file")
	xmp := []byte("<x:xmpmeta>from the png file</x:xmpmeta>")

	tests := []struct {
		name      string
		chunkType string
		data      []byte
		want      segment
		wantOK    bool
	}{
		{
			name:      "the exif chunk holds the exif data, which a jpeg segment starts with an identifier of",
			chunkType: exifChunkType,
			data:      exifBytes(),
			want:      segment{marker: app1Marker, payload: append([]byte(exifIdentifier), exifBytes()...)},
			wantOK:    true,
		},
		{
			name:      "the icc profile chunk holds a compressed profile, which a jpeg segment numbers",
			chunkType: iccProfileChunkType,
			data:      append([]byte("ICC Profile\x00\x00"), zlibBytes(t, profile)...),
			want: segment{
				marker:  app2Marker,
				payload: append([]byte(iccProfileIdentifier+"\x01"), profile...),
			},
			wantOK: true,
		},
		{
			name:      "the xmp text chunk holds the xmp metadata",
			chunkType: textChunkType,
			data:      append([]byte(xmpKeyword+"\x00"), xmp...),
			want:      segment{marker: app1Marker, payload: append([]byte(xmpIdentifier), xmp...)},
			wantOK:    true,
		},
		{
			name:      "the comment text chunk holds a comment",
			chunkType: textChunkType,
			data:      []byte(commentKeyword + "\x00written by gimp"),
			want:      segment{marker: commentMarker, payload: []byte("written by gimp")},
			wantOK:    true,
		},
		{
			name:      "a compressed comment text chunk holds a comment",
			chunkType: compressedChunkType,
			data:      append([]byte(commentKeyword+"\x00\x00"), zlibBytes(t, []byte("compressed comment"))...),
			want:      segment{marker: commentMarker, payload: []byte("compressed comment")},
			wantOK:    true,
		},
		{
			name:      "an international xmp chunk holds the xmp metadata",
			chunkType: internationalChunk,
			data:      append([]byte(xmpKeyword+"\x00\x00\x00\x00"), xmp...),
			want:      segment{marker: app1Marker, payload: append([]byte(xmpIdentifier), xmp...)},
			wantOK:    true,
		},
		{
			name:      "a compressed international comment chunk holds a comment",
			chunkType: internationalChunk,
			data: bytes.Join([][]byte{
				[]byte(commentKeyword + "\x00\x01en\x00translated comment\x00"),
				zlibBytes(t, []byte("comment in another language")),
			}, nil),
			want:   segment{marker: commentMarker, payload: []byte("comment in another language")},
			wantOK: true,
		},
		{
			name:      "a text chunk of another keyword holds no metadata of a jpeg file",
			chunkType: textChunkType,
			data:      []byte("Author\x00Bruce Wayne"),
			wantOK:    false,
		},
		{
			name:      "a text chunk with no keyword holds no metadata",
			chunkType: textChunkType,
			data:      []byte("no separator at all"),
			wantOK:    false,
		},
		{
			name:      "a text chunk with an empty text holds no metadata",
			chunkType: textChunkType,
			data:      []byte(commentKeyword + "\x00"),
			wantOK:    false,
		},
		{
			name:      "a compressed text chunk with no compressed text holds no metadata",
			chunkType: compressedChunkType,
			data:      []byte(commentKeyword + "\x00"),
			wantOK:    false,
		},
		{
			name:      "a compressed text chunk which is not compressed holds no metadata",
			chunkType: compressedChunkType,
			data:      append([]byte(commentKeyword+"\x00\x00"), profile...),
			wantOK:    false,
		},
		{
			name:      "an international chunk with no terminator holds no metadata",
			chunkType: internationalChunk,
			data:      []byte(commentKeyword + "\x00\x00en"),
			wantOK:    false,
		},
		{
			name:      "the pixels chunk holds no metadata",
			chunkType: "IDAT",
			data:      []byte("pixels"),
			wantOK:    false,
		},
		{
			name:      "the header chunk holds no metadata",
			chunkType: "IHDR",
			data:      ihdrData(100, 200),
			wantOK:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, ok := metadataChunk(test.chunkType, test.data)
			assert.Equal(t, test.wantOK, ok)

			if !test.wantOK {
				return
			}

			assert.Equal(t, test.want.marker, s.marker)
			assert.Equal(t, test.want.payload, s.payload)
		})
	}
}

// TestPNGMetadataSegments tells that the chunks of a PNG file are read until the
// end chunk, that the chunks of no interest are skipped, and that what follows
// the end chunk is not read.
func TestPNGMetadataSegments(t *testing.T) {
	file := pngBytes(
		pngChunk("IHDR", ihdrData(100, 200)),
		pngChunk(exifChunkType, exifBytes()),
		pngChunk("IDAT", []byte("pixels")),
		pngChunk(textChunkType, append([]byte(xmpKeyword+"\x00"), []byte("<x:xmpmeta>png</x:xmpmeta>")...)),
		pngChunk("IEND", nil),
		// A PNG file ends with the end chunk, and a file built by hand may hold
		// anything after it.
		[]byte("trailing bytes"),
	)

	segments, err := pngMetadataSegments(newReader(file))
	require.NoError(t, err)
	require.Len(t, segments, 2)

	assert.Equal(t, app1Marker, segments[0].marker)
	assert.Equal(t, exifIdentifier, string(segments[0].payload[:len(exifIdentifier)]))
	assert.Equal(t, app1Marker, segments[1].marker)
	assert.Equal(t, xmpIdentifier, string(segments[1].payload[:len(xmpIdentifier)]))
}

// TestPNGMetadataSegmentsNoEndChunk tells that the chunks read before a PNG file
// is cut short are kept, and that a file cut short in the middle of a chunk
// header is invalid.
func TestPNGMetadataSegmentsNoEndChunk(t *testing.T) {
	file := pngBytes(
		pngChunk(textChunkType, []byte(commentKeyword+"\x00comment of a truncated file")),
	)

	segments, err := pngMetadataSegments(newReader(file))
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, commentMarker, segments[0].marker)

	_, err = pngMetadataSegments(newReader([]byte(pngSignature + "\x00\x00")))
	assert.ErrorIs(t, err, ErrInvalidImage)
}

// TestPNGMetadataSegmentsErrors tells that a file which is not a PNG one, cut
// short in the middle of a chunk, or announcing more data than metadata takes,
// is told apart from a file holding no metadata.
func TestPNGMetadataSegmentsErrors(t *testing.T) {
	t.Run("a file with no signature is not a png file", func(t *testing.T) {
		file := pngBytes(pngChunk(textChunkType, []byte(commentKeyword+"\x00comment")))

		_, err := pngMetadataSegments(newReader(file[len(pngSignature):]))
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("a file shorter than the signature is invalid", func(t *testing.T) {
		_, err := pngMetadataSegments(newReader([]byte("\x89PNG\r\n")))
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	t.Run("a chunk announcing more data than metadata takes is not read", func(t *testing.T) {
		// The data of the chunk is not read, so the file cut short right after
		// the chunk header cannot even be skipped.
		chunk := binary.BigEndian.AppendUint32(nil, maxChunkDataSize+1)
		chunk = append(chunk, "IDAT"...)

		_, err := pngMetadataSegments(newReader([]byte(pngSignature + string(chunk))))
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	t.Run("a file with no chunk at all holds no metadata", func(t *testing.T) {
		segments, err := pngMetadataSegments(newReader(pngBytes(pngChunk("IEND", nil))))
		assert.NoError(t, err)
		assert.Empty(t, segments)
	})
}

// TestCopyFromPNG tells that the metadata chunks of a PNG file are written as
// the segments of the destination JPEG file, in the order a JPEG file wants,
// with the orientation and the dimensions of the destination file.
func TestCopyFromPNG(t *testing.T) {
	profile := []byte("icc profile of the png file")
	xmp := []byte("<x:xmpmeta>from the png file</x:xmpmeta>")

	srcPath := writeFixture(t, "source.png", pngBytes(
		pngChunk("IHDR", ihdrData(100, 200)),
		pngChunk(iccProfileChunkType, append([]byte("ICC Profile\x00\x00"), zlibBytes(t, profile)...)),
		pngChunk(exifChunkType, exifBytes()),
		pngChunk(textChunkType, append([]byte(xmpKeyword+"\x00"), xmp...)),
		pngChunk(textChunkType, []byte(commentKeyword+"\x00written by gimp")),
		pngChunk("IEND", nil),
	))

	dstData := jpeg(
		[]segment{
			{marker: commentMarker, payload: []byte("Lavc60.16.100")},
			startOfFrameSegment(1280, 1707),
		},
		startOfScanMarker,
		[]byte("pixels of the destination file"),
	)
	dstPath := writeFixture(t, "destination.jpg", dstData)

	require.NoError(t, Copy(dstPath, srcPath))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)

	header, endMarker, err := readHeader(newReader(data))
	require.NoError(t, err)
	assert.Equal(t, startOfScanMarker, endMarker)

	// The EXIF segment comes first, then the XMP, ICC and comment segments, the
	// comment of the source file replacing the one FFMPEG wrote, and the segment
	// of the destination file which holds no metadata is kept.
	require.Len(t, header, 5)
	assert.Equal(t, startOfFrameFirst, header[4].marker)

	assert.Equal(t, app1Marker, header[0].marker)
	structure, err := newTiff(header[0].payload[len(exifIdentifier):])
	require.NoError(t, err)
	// The pixels of the destination file are rotated by the encoder, and sized by
	// it, whatever the PNG file said.
	assert.Equal(t, orientationNormal,
		valueOf(t, structure, structure.firstIFDOffset(), orientationTag))
	assert.Equal(t, uint32(1280), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag))

	assert.Equal(t, app1Marker, header[1].marker)
	assert.Equal(t, xmpIdentifier, string(header[1].payload[:len(xmpIdentifier)]))
	assert.Equal(t, xmp, header[1].payload[len(xmpIdentifier):])

	assert.Equal(t, app2Marker, header[2].marker)
	assert.Equal(t, iccProfileIdentifier, string(header[2].payload[:len(iccProfileIdentifier)]))
	assert.Equal(t, profile, header[2].payload[len(iccProfileIdentifier)+1:])

	assert.Equal(t, commentMarker, header[3].marker)
	assert.Equal(t, []byte("written by gimp"), header[3].payload)

	assert.Contains(t, string(data), "pixels of the destination file")
	assert.NotContains(t, string(data), "Lavc60.16.100")

	// No temporary file is left behind next to the destination file.
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(dstPath), tempFilePattern))
	require.NoError(t, err)
	assert.Empty(t, leftovers)
}

// pngChunk returns the bytes of a chunk of a PNG file: the length of the data,
// the type of the chunk, the data, and the CRC of the type and of the data.
func pngChunk(chunkType string, data []byte) []byte {
	// A chunk of a PNG file holds no more than 31 bits of data.
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data))) //nolint:gosec
	chunk = append(chunk, chunkType...)
	chunk = append(chunk, data...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
}

// pngBytes returns the bytes of a PNG file holding the given chunks, behind the
// signature the file starts with.
func pngBytes(chunks ...[]byte) []byte {
	file := []byte(pngSignature)

	for _, chunk := range chunks {
		file = append(file, chunk...)
	}

	return file
}

// ihdrData returns the data of the header chunk of a PNG file, the chunk every
// one of them starts with, describing an image of the given dimensions.
func ihdrData(width, height uint32) []byte {
	data := binary.BigEndian.AppendUint32(nil, width)
	data = binary.BigEndian.AppendUint32(data, height)
	// Bit depth, color type, compression method, filter method and interlace.
	return append(data, 8, 2, 0, 0, 0)
}

// zlibBytes returns the data compressed with the zlib algorithm, which a PNG
// file uses to compress its ICC profile and some of its texts.
func zlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer

	writer := zlib.NewWriter(&buffer)
	_, err := writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buffer.Bytes()
}
