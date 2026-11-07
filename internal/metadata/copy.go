package metadata

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Errors about the image files read.
var (
	// ErrUnsupportedFormat is returned when an image file is not in a format
	// tinier knows how to read the metadata of, such as a HEIC or a WebP file.
	ErrUnsupportedFormat = errors.New("unsupported image format")
	// ErrInvalidImage is returned when an image file cannot be parsed.
	ErrInvalidImage = errors.New("invalid image file")
	// ErrInvalidExif is returned when the EXIF data of an image file cannot be
	// parsed.
	ErrInvalidExif = errors.New("invalid exif data")
)

const (
	// bufferSize is the size of the buffers used to read and to write the pixels
	// of a JPEG file.
	bufferSize = 128 * 1024
	// tempFilePattern is the name pattern of the temporary file which replaces
	// the destination file, in the directory of the latter.
	tempFilePattern = ".tinier-metadata-*"
)

// Copy copies the metadata of the srcPath image file into the dstPath JPEG
// file, and replaces the segments of the same kinds which the encoder of the
// destination file wrote. The metadata copied is made of the EXIF, XMP, ICC
// profile, IPTC and comment segments.
//
// The source file can be a JPEG, a PNG or an AVIF file, the three formats tinier
// converts. A PNG file keeps its metadata in chunks of its own, and an AVIF one
// in items of its own, both read and written back as the segments of a JPEG
// file. The chunks of the color space, of the pixel density and of the
// transparency, and the items of the pixels and of the properties of the image,
// which a JPEG file has no equivalent of, are left out.
//
// FFMPEG drops the metadata of the input file when it re-encodes an image, which
// is why the conversion loses it. The segments are transplanted as they are, and
// never re-serialized, which preserves the information FFMPEG cannot express,
// such as the maker notes, the GPS coordinates and the embedded thumbnail.
//
// The orientation and dimension tags of the copied EXIF are updated to describe
// the pixels of the destination file, because FFMPEG applies the rotation of
// the orientation tag to the pixels it writes. An EXIF segment which cannot be
// updated is dropped, rather than kept to describe the pixels of the input
// file.
//
// The destination file is replaced as a whole, so it is left untouched when it
// cannot be written, as a file of a read only directory or of a full disk does.
//
// It returns an error matching ErrUnsupportedFormat when one of the files is
// neither a JPEG file nor, for the source file, a PNG one, and no error at all,
// copying nothing, when the source file holds no metadata.
func Copy(dstPath, srcPath string) error {
	metadata, err := readMetadata(srcPath)
	if err != nil {
		return err
	} else if len(metadata) == 0 {
		return nil
	}

	file, err := os.Open(dstPath)
	if err != nil {
		return fmt.Errorf("opening the destination file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	reader := bufio.NewReaderSize(file, bufferSize)
	header, endMarker, err := readHeader(reader)
	if err != nil {
		return fmt.Errorf("reading the header of the destination file: %w", err)
	}

	width, height := imageDimensions(header)
	metadata = reconcileMetadata(metadata, width, height)
	if len(metadata) == 0 {
		return nil
	}

	kept := keepSegments(header, metadata)

	tmpPath, err := writeTransplanted(dstPath, metadata, kept, endMarker, reader)
	if err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing the destination file: %w", err)
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing the destination file: %w", err)
	}

	return nil
}

// formatLength is the number of bytes telling the format of an image file apart,
// at the start of the file. It reaches the brand of an AVIF file, which follows
// the length and the name of the box telling the type of the file.
const formatLength = 12

// readMetadata returns the metadata segments of the srcPath image file, in the
// order they have to be written to a destination file. The bytes at the start of
// the file tell which reader to use: a JPEG file keeps its metadata in segments,
// and a PNG file in chunks.
func readMetadata(srcPath string) ([]segment, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("opening the source file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	reader := bufio.NewReaderSize(file, bufferSize)

	start, err := reader.Peek(formatLength)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrUnsupportedFormat
		}
		return nil, err
	}

	segments, err := formatSegments(start, reader, file)
	if err != nil {
		return nil, err
	}

	// The payloads of the metadata segments are updated in place later on, to
	// describe the pixels of the destination file. They are the freshly
	// allocated ones of the readers, so nothing else points at them.
	return sortSegments(segments), nil
}

// formatSegments returns the metadata segments of a file, read the way the
// format the given bytes, which start the file, tell it is written in does.
func formatSegments(start []byte, reader *bufio.Reader, file *os.File) ([]segment, error) {
	switch {
	case bytes.HasPrefix(start, []byte{markerPrefix, startOfImageMarker}):
		header, _, err := readHeader(reader)
		if err != nil {
			return nil, fmt.Errorf("reading the segments of the source file: %w", err)
		}
		return metadataSegments(header), nil

	case bytes.HasPrefix(start, []byte(pngSignature)):
		segments, err := pngMetadataSegments(reader)
		if err != nil {
			return nil, fmt.Errorf("reading the chunks of the source file: %w", err)
		}
		return segments, nil

	case isAVIF(start):
		info, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("reading the size of the source file: %w", err)
		}
		segments, err := avifMetadataSegments(file, info.Size())
		if err != nil {
			return nil, fmt.Errorf("reading the items of the source file: %w", err)
		}
		return segments, nil

	default:
		return nil, fmt.Errorf("%w: only JPEG, PNG and AVIF files hold metadata",
			ErrUnsupportedFormat)
	}
}

// metadataSegments returns the segments which carry metadata, in the order they
// appear in the file.
func metadataSegments(header []segment) []segment {
	var metadata []segment

	for _, s := range header {
		if s.kind() != kindOther {
			metadata = append(metadata, s)
		}
	}

	return metadata
}

// sortSegments returns the segments sorted by kind, in the order they have to
// be written to a destination file. The segments given are not modified.
func sortSegments(segments []segment) []segment {
	sorted := make([]segment, 0, len(segments))

	for _, k := range copyOrder {
		seen := 0

		for _, s := range segments {
			if s.kind() != k {
				continue
			}

			seen++

			// An ICC profile is split into as many segments as its writer needs,
			// and a file can hold more than one comment. The other kinds of
			// metadata appear once, and a file holding two of them is malformed.
			if seen > 1 && k != kindICCProfile && k != kindComment {
				continue
			}

			sorted = append(sorted, s)
		}
	}

	return sorted
}

// keepSegments returns the segments of a destination file which are not replaced
// by copied metadata, in the order they appear in the file.
func keepSegments(header []segment, metadata []segment) []segment {
	replaced := replacedKinds(metadata)

	var kept []segment

	for _, s := range header {
		if k := s.kind(); k == kindOther || !replaced[k] {
			kept = append(kept, s)
		}
	}

	return kept
}

// replacedKinds returns the set of kinds of metadata the segments hold.
func replacedKinds(metadata []segment) map[kind]bool {
	kinds := make(map[kind]bool, len(metadata))

	for _, s := range metadata {
		kinds[s.kind()] = true
	}

	return kinds
}

// imageDimensions returns the dimensions of the first frame of an image, which
// are the dimensions of its pixels. Both are zero when the image holds no start
// of frame segment.
func imageDimensions(header []segment) (width, height uint32) {
	for _, s := range header {
		if w, h, ok := s.frameSize(); ok {
			return w, h
		}
	}

	return 0, 0
}

// reconcileMetadata returns the metadata describing the pixels of the
// destination file, sized width by height. The orientation and dimension tags
// of the EXIF segments are updated in place, and an EXIF segment which cannot
// be updated is dropped. A frame of zero dimension leaves the dimensions
// untouched.
//
// FFMPEG applies the rotation of the orientation tag to the pixels of the file
// it writes, and does not tell it apart from a rotation which was already in
// the pixels of the input file. The tag is therefore always set to normal: the
// rotation is in the pixels, and describing them as rotated would rotate them
// a second time.
func reconcileMetadata(metadata []segment, width, height uint32) []segment {
	reconciled := make([]segment, 0, len(metadata))

	for _, s := range metadata {
		if s.kind() != kindExif {
			reconciled = append(reconciled, s)
			continue
		}

		// The payload of the segment shares the memory of the data the tiff
		// structure updates, so the segment holds the updated EXIF.
		structure, err := newTiff(s.payload[len(exifIdentifier):])
		if err != nil {
			// A structure which cannot be read still describes the pixels of the
			// input file, and would state a rotation and dimensions which the
			// pixels of the destination file do not have.
			continue
		}

		structure.setOrientationNormal()
		structure.setDimensions(width, height)

		reconciled = append(reconciled, s)
	}

	return reconciled
}

// writeTransplanted writes the metadata segments, the kept segments and the rest
// of the destination file into a temporary file, located in the directory of
// dstPath, and returns its path. The permissions of the destination file are
// applied to it, so it can replace the destination file.
func writeTransplanted(
	dstPath string,
	metadata, kept []segment,
	endMarker uint8,
	rest io.Reader,
) (string, error) {
	info, err := os.Stat(dstPath)
	if err != nil {
		return "", fmt.Errorf("reading the permissions of the destination file: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(dstPath), tempFilePattern)
	if err != nil {
		return "", fmt.Errorf("creating a temporary file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// The temporary file is removed when it cannot replace the destination
	// file, whatever the step which failed to be.
	defer func() {
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	err = writeSegments(tmpFile, metadata, kept, endMarker, rest)
	if err != nil {
		return "", err
	}

	if err = tmpFile.Chmod(info.Mode().Perm()); err != nil {
		return "", fmt.Errorf("setting the permissions of the temporary file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return "", fmt.Errorf("closing the temporary file: %w", err)
	}

	return tmpPath, nil
}

// writeSegments writes the header of a JPEG file, made of the start of image
// marker, of the metadata segments and of the kept segments, followed by the
// marker ending the header and by the rest of the file.
func writeSegments(
	writer io.Writer,
	metadata, kept []segment,
	endMarker uint8,
	rest io.Reader,
) error {
	buffered := bufio.NewWriterSize(writer, bufferSize)

	// The segments of the input file are written before the ones of the
	// destination file which are kept, so the EXIF segment directly follows the
	// start of image marker, as the EXIF standard requires it to.
	segments := make([]segment, 0, len(metadata)+len(kept))
	segments = append(segments, metadata...)
	segments = append(segments, kept...)

	header := make([]byte, 0, segmentsSize(metadata, kept)+2*markerSize)
	header = append(header, markerPrefix, startOfImageMarker)

	for _, s := range segments {
		header = append(header, s.appended()...)
	}

	header = append(header, markerPrefix, endMarker)

	if _, err := buffered.Write(header); err != nil {
		return fmt.Errorf("writing the header: %w", err)
	}

	if _, err := io.Copy(buffered, rest); err != nil {
		return fmt.Errorf("writing the pixels: %w", err)
	}

	if err := buffered.Flush(); err != nil {
		return fmt.Errorf("flushing the buffer: %w", err)
	}

	return nil
}

// segmentsSize returns the number of bytes the segments take in a JPEG file.
func segmentsSize(segments ...[]segment) int {
	size := 0

	for _, group := range segments {
		for _, s := range group {
			size += s.size()
		}
	}

	return size
}
