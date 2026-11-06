// Package metadata copies the metadata of an image file, such as the EXIF data, into
// a JPEG file.
//
// FFMPEG writes no metadata when it re-encodes an image, therefore the metadata
// is transplanted from the input file once the conversion is done, without being
// re-serialized. This preserves information FFMPEG cannot express, such as maker
// notes, GPS coordinates and thumbnails.
//
// The metadata of a JPEG or PNG input file can be read, and it can only be
// written to a JPEG output file: an AVIF or HEIC input file, and a non-JPEG
// output file, are both unsupported.
package metadata
