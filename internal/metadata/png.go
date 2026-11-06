package metadata

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// pngSignature is the sequence of bytes every PNG file starts with.
const pngSignature = "\x89PNG\r\n\x1a\n"

// Types of the chunks of a PNG file, each one made of 4 ASCII characters. Only
// the chunks holding metadata are named here, the others are skipped.
const (
	iendChunkType       = "IEND"
	exifChunkType       = "eXIf"
	iccProfileChunkType = "iCCP"
	textChunkType       = "tEXt"
	compressedChunkType = "zTXt"
	internationalChunk  = "iTXt"
)

// Keywords of the text chunks of a PNG file, telling which metadata they hold.
const (
	commentKeyword = "Comment"
	xmpKeyword     = "XML:com.adobe.xmp"
)

// Lengths of the fields of a chunk: the length and the type precede its data,
// and the CRC of the chunk follows it.
const (
	chunkLengthSize = 4
	chunkTypeSize   = 4
	chunkCRCSize    = 4
	// maxChunkDataSize is above the size of the largest ICC profile and of the
	// largest XMP packet a PNG file holds, so a chunk announcing more data is
	// not metadata tinier has to read.
	maxChunkDataSize = 1 << 24
)

// pngMetadataSegments returns the metadata held by the chunks of a PNG file, as
// the segments of the JPEG file holding the same metadata. A PNG file keeps its
// EXIF data, its ICC profile and its XMP metadata in chunks of its own, so the
// segments built here carry the identifiers a JPEG file expects at the start of
// the payload of a segment.
//
// Only the chunks a JPEG file has an equivalent of are returned. The others,
// such as the ones of the color space, of the pixel density and of the
// transparency, have no equivalent in a JPEG file, and are skipped. So is a
// chunk tinier cannot read, which leaves the conversion go on with the
// metadata, if any, the other chunks hold.
func pngMetadataSegments(reader io.Reader) ([]segment, error) {
	signature := make([]byte, len(pngSignature))
	if _, err := io.ReadFull(reader, signature); err != nil {
		return nil, fmt.Errorf("%w: reading the signature: %w", ErrInvalidImage, err)
	} else if string(signature) != pngSignature {
		return nil, ErrUnsupportedFormat
	}

	var metadata []segment

	for {
		chunkType, data, err := readChunk(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// A file with no end chunk is truncated, but the chunks read so
				// far are worth keeping.
				return metadata, nil
			}
			return nil, err
		} else if chunkType == iendChunkType {
			return metadata, nil
		}

		s, ok := metadataChunk(chunkType, data)
		if ok {
			metadata = append(metadata, s)
		}
	}
}

// readChunk returns the type and the data of the next chunk of a PNG file. It
// returns io.EOF when the file holds no more chunk.
//
// The CRC which ends a chunk is read to be skipped: it is not needed to reach
// the next chunk, and a corrupted chunk holds no metadata worth failing the
// conversion for.
func readChunk(reader io.Reader) (chunkType string, data []byte, err error) {
	var header [chunkLengthSize + chunkTypeSize]byte

	if _, err = io.ReadFull(reader, header[:]); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return "", nil, fmt.Errorf("%w: truncated header of a chunk: %w", ErrInvalidImage, err)
		}
		return "", nil, err
	}

	length := binary.BigEndian.Uint32(header[:chunkLengthSize])
	chunkType = string(header[chunkLengthSize:])

	if length > maxChunkDataSize {
		// The data is too big to be metadata, and reading it would allocate the
		// memory of a file which is not a PNG one.
		if err = skip(reader, int64(length)+chunkCRCSize); err != nil {
			// Not an io.EOF: the chunk announces data the file does not hold, which
			// is a truncated file, not the end of it.
			return "", nil, fmt.Errorf("%w: the data of the chunk %q is truncated", ErrInvalidImage, chunkType)
		}
		return chunkType, nil, nil
	}

	data = make([]byte, length)
	if _, err = io.ReadFull(reader, data); err != nil {
		return "", nil, fmt.Errorf("%w: truncated data of the chunk %q: %w",
			ErrInvalidImage, chunkType, err)
	}

	if err = skip(reader, chunkCRCSize); err != nil {
		return "", nil, fmt.Errorf("%w: skipping the CRC of the chunk %q: %w",
			ErrInvalidImage, chunkType, err)
	}

	return chunkType, data, nil
}

// skip discards the next n bytes of a reader.
func skip(reader io.Reader, n int64) error {
	_, err := io.CopyN(io.Discard, reader, n)
	return err
}

// metadataChunk returns the segment of a JPEG file holding the metadata of the
// PNG chunk of the given type and data, and reports whether the chunk holds
// metadata a JPEG file can express.
func metadataChunk(chunkType string, data []byte) (s segment, ok bool) {
	switch chunkType {
	case exifChunkType:
		// A PNG file holds the EXIF data with no identifier in front of it, when
		// a JPEG file starts the payload of its segment with one.
		return segment{
			marker:  app1Marker,
			payload: withIdentifier(data, exifIdentifier),
		}, true

	case iccProfileChunkType:
		profile := iccProfile(data)
		if profile == nil {
			return segment{}, false
		}
		// A JPEG file numbers the segments of an ICC profile, starting at one,
		// between the identifier and the profile.
		return segment{
			marker:  app2Marker,
			payload: withIdentifier(profile, iccProfileIdentifier+"\x01"),
		}, true

	case textChunkType:
		return textSegment(textOf(data))

	case compressedChunkType:
		keyword, text := textOf(data)
		if method, compressed, found := bytes.Cut(text, []byte{0}); found {
			// The compression method byte tells nothing: the only one defined is
			// the zlib one, which is the one used.
			_ = method
			text = inflatedData(compressed)
		} else {
			keyword, text = "", nil
		}
		return textSegment(keyword, text)

	case internationalChunk:
		keyword, text := internationalText(data)
		return textSegment(keyword, text)
	}

	return segment{}, false
}

// textOf returns the keyword and the text of a text chunk, separated by a null
// byte. Both are empty when the chunk holds none.
func textOf(data []byte) (keyword string, text []byte) {
	keywordBytes, text, found := bytes.Cut(data, []byte{0})
	if !found {
		return "", nil
	}
	return string(keywordBytes), text
}

// internationalText returns the keyword and the text of an iTXt chunk, made of
// a keyword, a compression flag telling if the text is compressed with zlib, a
// language tag and a translated keyword, all separated by null bytes, and of
// the text which ends the chunk. The keyword and the text are empty when the
// chunk holds none.
func internationalText(data []byte) (keyword string, text []byte) {
	keywordBytes, rest, found := bytes.Cut(data, []byte{0})
	if !found {
		return "", nil
	}

	if len(rest) < 1 {
		return "", nil
	}

	compressed := rest[0] == 1

	// The language tag and the translated keyword are of no use here, and both
	// are terminated by a null byte.
	_, rest, found = bytes.Cut(rest[1:], []byte{0})
	if !found {
		return "", nil
	}
	_, rest, found = bytes.Cut(rest, []byte{0})
	if !found {
		return "", nil
	}

	if compressed {
		return string(keywordBytes), inflatedData(rest)
	}

	return string(keywordBytes), rest
}

// textSegment returns the segment of a JPEG file holding the metadata the text
// chunk of the given keyword describes, and reports whether the keyword
// describes metadata a JPEG file can express.
func textSegment(keyword string, text []byte) (segment, bool) {
	if len(text) == 0 {
		return segment{}, false
	}

	switch keyword {
	case xmpKeyword:
		return segment{
			marker:  app1Marker,
			payload: withIdentifier(text, xmpIdentifier),
		}, true

	case commentKeyword:
		return segment{marker: commentMarker, payload: text}, true
	}

	return segment{}, false
}

// iccProfile returns the profile held by a PNG iCCP chunk, made of the name of
// the profile, a null byte, the compression method and the zlib compressed
// profile. It returns nil when the chunk holds no profile.
func iccProfile(data []byte) []byte {
	// The name of the profile tells nothing: a JPEG file holds one profile too,
	// and does not name it.
	_, rest, found := bytes.Cut(data, []byte{0})
	if !found || len(rest) < 1 {
		return nil
	}
	return inflatedData(rest[1:])
}

// inflatedData returns the data compressed with the zlib algorithm, which the
// PNG format uses to compress the ICC profile and some of its texts. It returns
// nil when the data is not compressed as it says it is.
func inflatedData(data []byte) []byte {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	defer func() {
		_ = reader.Close()
	}()

	inflated, err := io.ReadAll(reader)
	if err != nil {
		return nil
	}

	return inflated
}

// withIdentifier returns the payload of a JPEG segment holding the given data,
// written after the identifier of the segment.
func withIdentifier(data []byte, identifier string) []byte {
	payload := make([]byte, 0, len(identifier)+len(data))
	payload = append(payload, identifier...)
	return append(payload, data...)
}
