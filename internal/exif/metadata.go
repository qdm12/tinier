package exif

import "bytes"

// Identifiers of the metadata, found at the start of the payload of a segment.
const (
	xmpIdentifier         = "http://ns.adobe.com/xap/1.0/\x00"
	xmpExtendedIdentifier = "http://ns.adobe.com/xmp/extension/\x00"
	iccProfileIdentifier  = "ICC_PROFILE\x00"
	photoshopIdentifier   = "Photoshop 3.0\x00"
	iptcRawIdentifier     = "\x1C\x02\x00"
)

// kind is the kind of metadata a segment carries. It tells which segments are
// copied from the input file, and which segments of the output file they
// replace.
type kind int

const (
	// kindOther is the kind of the segments which carry no metadata, such as the
	// quantization tables and the start of frame segment.
	kindOther kind = iota
	kindExif
	kindXMP
	kindICCProfile
	kindIPTC
	kindComment
)

// copyOrder is the order in which the kinds of metadata are written to the
// output file. The EXIF segment comes first, as the EXIF standard requires it
// to be the segment which directly follows the start of image marker. The other
// kinds follow in the order recommended by the Exif, XMP and ICC standards.
//
//nolint:gochecknoglobals // order of a fixed set, never mutated
var copyOrder = []kind{kindExif, kindXMP, kindICCProfile, kindIPTC, kindComment}

// kind returns the kind of metadata carried by the segment.
func (s segment) kind() kind {
	switch s.marker {
	case app1Marker:
		switch {
		case startsWith(s.payload, exifIdentifier):
			return kindExif
		case startsWith(s.payload, xmpIdentifier),
			startsWith(s.payload, xmpExtendedIdentifier):
			return kindXMP
		}
	case app2Marker:
		if startsWith(s.payload, iccProfileIdentifier) {
			return kindICCProfile
		}
	case app13Marker:
		if startsWith(s.payload, photoshopIdentifier) ||
			startsWith(s.payload, iptcRawIdentifier) {
			return kindIPTC
		}
	case commentMarker:
		return kindComment
	}

	return kindOther
}

// startsWith reports whether a payload starts with the given identifier.
func startsWith(payload []byte, identifier string) bool {
	return bytes.HasPrefix(payload, []byte(identifier))
}
