package metadata

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The AVIF files of the test data are two real files, of the test pattern of
// FFMPEG: one written by libavif, the library behind sharp and Chrome, which
// holds an EXIF structure and a profile of colors, and one written by FFMPEG,
// which holds neither. They are the only way to check that what tinier reads is
// what a file really holds.
const (
	avifWithMetadata    = "testdata/encoded-by-libavif.avif"
	avifWithoutMetadata = "testdata/encoded-by-ffmpeg.avif"
)

func TestIsAVIF(t *testing.T) {
	tests := []struct {
		name  string
		start []byte
		want  bool
	}{
		{"an avif file", []byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00mif1miaf"), true},
		{"an avif file of the av01 brand", []byte("\x00\x00\x00\x18ftypav01\x00\x00\x00\x00mif1miaf"), true},
		{"an animated avif file", []byte("\x00\x00\x00\x18ftypavis\x00\x00\x00\x00mif1miaf"), true},
		{"a heic file", []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1miaf"), false},
		{"another brand of an avif file", []byte("\x00\x00\x00\x18ftypmif1\x00\x00\x00\x00avif"), false},
		{"a jpeg file", []byte{0xFF, 0xD8, 0xFF, 0xE0, 'A', 'd', 'o', 'b', 'e'}, false},
		{"a png file", []byte(pngSignature), false},
		{"fewer bytes than the brand", []byte("\x00\x00\x00\x0cftypavi"), false},
		{"an empty file", nil, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isAVIF(test.start))
		})
	}
}

// TestAVIFMetadata reads the metadata of a real AVIF file, and checks it against
// what was put into it: an EXIF structure of a picture of 1707 by 1280 pixels,
// of a maker of Wayne Enterprises, and a profile of colors.
func TestAVIFMetadata(t *testing.T) {
	segments, err := readMetadata(avifWithMetadata)
	require.NoError(t, err)

	markers := make([]uint8, 0, len(segments))
	for _, s := range segments {
		markers = append(markers, s.marker)
	}

	assert.Equal(t, []uint8{app1Marker, app2Marker}, markers,
		"an application segment of the EXIF structure and one of the profile of colors")

	exifSegment := segmentsOfKind(segments, kindExif)
	require.Len(t, exifSegment, 1)

	structure, err := newTiff(bytes.TrimPrefix(exifSegment[0].payload,
		[]byte(exifIdentifier)))
	require.NoError(t, err)

	assert.Equal(t, uint32(1707), valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag))
	assert.Equal(t, uint32(1280), valueOf(t, structure, structure.firstIFDOffset(), imageLengthTag))
	assert.Equal(t, orientationNormal,
		valueOf(t, structure, structure.firstIFDOffset(), orientationTag))

	profiles := segmentsOfKind(segments, kindICCProfile)
	require.Len(t, profiles, 1)
	assert.Equal(t, app2Marker, profiles[0].marker)
	assert.True(t, startsWith(profiles[0].payload, iccProfileIdentifier),
		"the profile of colors of the file is an Internet Content Contributor profile")
	// The profile of colors of a PNG file and one of an AVIF file are held the
	// same way, in the box of the properties of the items.
	assert.Greater(t, len(profiles[0].payload), len(iccProfileIdentifier)+2,
		"the profile of colors is not empty")
}

// TestAVIFMetadataWithoutMetadata checks a real AVIF file which holds no
// metadata: the items of its pixels are not metadata, and the profile of colors
// it describes by parameters is not one either.
func TestAVIFMetadataWithoutMetadata(t *testing.T) {
	segments, err := readMetadata(avifWithoutMetadata)
	require.NoError(t, err)
	assert.Empty(t, segments)
}

func TestAVIFMetadataOfTruncatedFile(t *testing.T) {
	data, err := os.ReadFile(avifWithMetadata)
	require.NoError(t, err)

	// Each of the cuts leaves the file shorter than the boxes it announces.
	for _, keep := range []int{2, 8, 16, 64, 512, len(data) - 1} {
		t.Run(fmt.Sprintf("cut at %d bytes", keep), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cut.avif")
			// The path is a file of the temporary directory of the test.
			require.NoError(t, os.WriteFile(path, data[:keep], 0600)) //nolint:gosec

			// A file cut short is either read as holding no metadata, or refused.
			// What it cannot be is metadata half read: a file too short to even
			// tell the format of is refused as an unsupported format, and one
			// whose boxes announce more bytes than it holds as invalid.
			segments, err := readMetadata(path)
			if err != nil {
				refused := errors.Is(err, ErrInvalidImage) || errors.Is(err, ErrUnsupportedFormat)
				assert.True(t, refused, "got %v", err)
				return
			}

			for _, s := range segmentsOfKind(segments, kindExif) {
				assert.Greater(t, len(s.payload), len(exifIdentifier))
			}
		})
	}
}

// TestCopyFromAVIF writes the metadata of a real AVIF file into a JPEG file, as
// tinier does after it converted the file.
func TestCopyFromAVIF(t *testing.T) {
	// A JPEG file as FFMPEG would leave it: a start of frame of 640 by 480
	// pixels, and nothing else.
	destination := startOfFrameSegment(640, 480)
	path := filepath.Join(t.TempDir(), "converted.jpg")
	require.NoError(t, os.WriteFile(path, append([]byte{markerPrefix, startOfImageMarker},
		append(destination.appended(), markerPrefix, startOfScanMarker, markerPrefix, endOfImageMarker)...),
		0600))

	require.NoError(t, Copy(path, avifWithMetadata))

	file, err := os.Open(path)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, file.Close())
	}()

	header, _, err := readHeader(bufio.NewReader(file))
	require.NoError(t, err)

	assert.Equal(t, []uint8{app1Marker, app2Marker, startOfFrameFirst}, []uint8{
		header[0].marker, header[1].marker, header[2].marker,
	}, "the segments of the metadata of the AVIF file come before the pixels")

	structure, err := newTiff(bytes.TrimPrefix(segmentsOfKind(header, kindExif)[0].payload,
		[]byte(exifIdentifier)))
	require.NoError(t, err)

	// The dimensions of the EXIF structure are those of the new pixels, and not
	// those of the picture the file was made from.
	assert.Equal(t, uint32(640),
		valueOf(t, structure, structure.firstIFDOffset(), imageWidthTag))
	assert.Equal(t, uint32(480),
		valueOf(t, structure, structure.firstIFDOffset(), imageLengthTag))

	profile := segmentsOfKind(header, kindICCProfile)[0].payload
	assert.True(t, startsWith(profile, iccProfileIdentifier))
}

// TestAVIFMetadataOfOtherFormats checks the formats which are not AVIF: their
// metadata is read elsewhere, and an AVIF reader handed one of them has to say
// so rather than fail.
func TestAVIFMetadataOfOtherFormats(t *testing.T) {
	t.Run("a file shorter than a box is refused", func(t *testing.T) {
		_, err := avifMetadataSegments(bytes.NewReader([]byte{0, 0}), 2)
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	// A file of another format never reaches this reader: the format of a file
	// tells which reader takes it, and TestIsAVIF checks that a JPEG or a PNG
	// file is left to the readers of tinier.
}

// segmentsOfKind returns the segments of the given kind of metadata.
func segmentsOfKind(segments []segment, want kind) []segment {
	var found []segment

	for _, s := range segments {
		if s.kind() == want {
			found = append(found, s)
		}
	}

	return found
}

// An AVIF file of the test data holds no XMP packet, and writing one needs a
// writer of the boxes of a base media file: the item of the packet is therefore
// read here on its own, next to the item of the EXIF structure.
func TestItemSegment(t *testing.T) {
	t.Run("an item of an exif structure", func(t *testing.T) {
		structure := exifBytes()
		// The four bytes which start the item tell how far from them, and from
		// the four, the EXIF structure starts.
		contents := binary.BigEndian.AppendUint32(nil, 0)
		contents = append(contents, structure...)

		s, err := itemSegment(bytes.NewReader(contents),
			item{identifier: 1, name: exifItemName},
			place{offset: 0, length: int64(len(contents))})
		require.NoError(t, err)
		assert.Equal(t, app1Marker, s.marker)
		assert.True(t, startsWith(s.payload, exifIdentifier),
			"the payload of the segment is the structure, behind its identifier")
		assert.Equal(t, len(exifIdentifier)+len(structure), len(s.payload))
	})

	t.Run("an item of an xmp packet", func(t *testing.T) {
		// The item of an XMP packet is the packet itself, which the item of an
		// EXIF structure is not: the four bytes of an offset precede it.
		packet := []byte("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\"></x:xmpmeta>")

		s, err := itemSegment(bytes.NewReader(packet),
			item{identifier: 2, name: xmpItemName},
			place{offset: 0, length: int64(len(packet))})
		require.NoError(t, err)
		assert.Equal(t, app1Marker, s.marker)
		assert.Equal(t, string(withIdentifier(packet, xmpIdentifier)), string(s.payload))
	})

	t.Run("an item of the pixels is no segment", func(t *testing.T) {
		contents := []byte("the pixels of an image")

		s, err := itemSegment(bytes.NewReader(contents),
			item{identifier: 3, name: "avat"},
			place{offset: 0, length: int64(len(contents))})
		require.NoError(t, err)
		assert.Equal(t, segment{}, s, "a JPEG file has no equivalent of the pixels of an AVIF file")
	})

	t.Run("an item longer than metadata can be", func(t *testing.T) {
		_, err := itemSegment(bytes.NewReader([]byte("short")),
			item{identifier: 1, name: exifItemName},
			place{offset: 0, length: maxMetadataBoxLength + 1})
		assert.ErrorIs(t, err, ErrInvalidImage)
	})

	t.Run("an item the file does not hold", func(t *testing.T) {
		_, err := itemSegment(bytes.NewReader([]byte("short")),
			item{identifier: 1, name: exifItemName},
			place{offset: 0, length: 100})
		assert.ErrorIs(t, err, ErrInvalidImage)
	})
}
