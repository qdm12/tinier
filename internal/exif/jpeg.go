package exif

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// JPEG markers, the byte which follows the 0xFF prefix byte.
const (
	startOfImageMarker uint8 = 0xD8
	endOfImageMarker   uint8 = 0xD9
	startOfScanMarker  uint8 = 0xDA

	restartMarkerFirst uint8 = 0xD0
	restartMarkerLast  uint8 = 0xD7

	app1Marker  uint8 = 0xE1
	app2Marker  uint8 = 0xE2
	app13Marker uint8 = 0xED

	commentMarker uint8 = 0xFE

	// startOfFrameFirst and startOfFrameLast bound the range of the markers of
	// the segments holding the dimensions of the image. The markers within the
	// range, which are not start of frame markers, are named after.
	startOfFrameFirst      uint8 = 0xC0
	startOfFrameLast       uint8 = 0xCF
	huffmanTableMarker     uint8 = 0xC4
	losslessFrameMarker    uint8 = 0xC8
	conditionalTableMarker uint8 = 0xCC

	// markerPrefix starts every marker, and is also used as a fill byte. A
	// marker prefix followed by a 0x00 byte is a padding byte.
	markerPrefix uint8 = 0xFF
	paddingByte  uint8 = 0x00
)

const (
	// lengthSize is the size of the length field of a segment, which counts its
	// own bytes.
	lengthSize = 2
	// segmentSize is the number of bytes a segment takes besides its payload: the
	// prefix and marker bytes, and the length field.
	segmentSize = 2 + lengthSize
	// markerSize is the size of a marker which holds no length nor payload, such as
	// the start of image marker and the ones which end the header.
	markerSize = 2
)

// segment is a segment of a JPEG file: a marker and the payload which follows
// it. The length field of the segment is not part of the payload.
type segment struct {
	marker  uint8
	payload []byte
}

// size returns the number of bytes the segment takes in a JPEG file, with its
// marker and its length field.
func (s segment) size() int {
	return segmentSize + len(s.payload)
}

// appended returns the segment as it is stored in a JPEG file, with its marker
// and its length field.
func (s segment) appended() []byte {
	segmentBytes := make([]byte, 0, s.size())
	segmentBytes = append(segmentBytes, markerPrefix, s.marker)
	// The JPEG format limits a payload to 65533 bytes, so the size of the
	// segment fits in the 2 bytes of its length field.
	size := uint16(len(s.payload) + lengthSize) //nolint:gosec
	segmentBytes = binary.BigEndian.AppendUint16(segmentBytes, size)
	return append(segmentBytes, s.payload...)
}

// frameSize returns the dimensions of the image held by a start of frame
// segment, and reports whether the segment is one.
func (s segment) frameSize() (width, height uint32, ok bool) {
	if !isStartOfFrame(s.marker) || len(s.payload) < 5 {
		return 0, 0, false
	}

	// The payload holds the sample precision, then the height and the width of
	// the image, both on 2 bytes.
	return uint32(binary.BigEndian.Uint16(s.payload[3:5])),
		uint32(binary.BigEndian.Uint16(s.payload[1:3])), true
}

// isStartOfFrame reports whether a marker is the marker of a segment holding
// the dimensions of the image.
func isStartOfFrame(marker uint8) bool {
	if marker < startOfFrameFirst || marker > startOfFrameLast {
		return false
	}

	return marker != huffmanTableMarker &&
		marker != losslessFrameMarker &&
		marker != conditionalTableMarker
}

// readHeader reads the start of image marker and the segments which follow it,
// up to the marker which ends the header. The reader is left after the returned
// end marker, so the rest of the file can be streamed as it is.
//
// It returns ErrUnsupportedFormat if the file is not a JPEG file, and ErrInvalidImage if
// it cannot be parsed.
func readHeader(reader *bufio.Reader) (segments []segment, endMarker uint8, err error) {
	marker, err := readMarker(reader)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, 0, fmt.Errorf("%w: the file holds no marker", ErrUnsupportedFormat)
		}
		return nil, 0, err
	} else if marker != startOfImageMarker {
		return nil, 0, fmt.Errorf("%w: expected the marker %#X, got %#X",
			ErrUnsupportedFormat, startOfImageMarker, marker)
	}

	for {
		marker, err := readMarker(reader)
		if err != nil {
			return nil, 0, err
		}
		if endsHeader(marker) {
			return segments, marker, nil
		}
		payload, err := readPayload(reader, marker)
		if err != nil {
			return nil, 0, err
		}
		segments = append(segments, segment{marker: marker, payload: payload})
	}
}

// endsHeader reports whether a marker ends the header of a JPEG file. The
// markers which do are the start of scan marker, announcing the compressed image
// data, the end of image marker, and the restart markers. None of them is
// followed by the length of a payload.
func endsHeader(marker uint8) bool {
	if marker == startOfScanMarker || marker == endOfImageMarker {
		return true
	}

	return marker >= restartMarkerFirst && marker <= restartMarkerLast
}

// readMarker reads a marker and returns it without the prefix byte which starts
// it. The fill and padding bytes which precede it are skipped.
func readMarker(reader *bufio.Reader) (uint8, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return 0, err
	} else if b != markerPrefix {
		return 0, fmt.Errorf("%w: expected %#X, got %#X", ErrUnsupportedFormat, markerPrefix, b)
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		} else if b != markerPrefix && b != paddingByte {
			return b, nil
		}
	}
}

// readPayload reads the payload of a segment, which is announced by the length
// field following the given marker.
func readPayload(reader *bufio.Reader, marker uint8) ([]byte, error) {
	var lengthBytes [2]byte
	if _, err := io.ReadFull(reader, lengthBytes[:]); err != nil {
		return nil, fmt.Errorf("%w: reading the length of the segment %#X", ErrInvalidImage, marker)
	}

	// The length of a segment counts its own 2 bytes.
	length := binary.BigEndian.Uint16(lengthBytes[:])
	if length < lengthSize {
		return nil, fmt.Errorf("%w: the segment %#X announces a length of %d",
			ErrInvalidImage, marker, length)
	}

	payload := make([]byte, length-lengthSize)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("%w: reading the payload of the segment %#X", ErrInvalidImage, marker)
	}

	return payload, nil
}
