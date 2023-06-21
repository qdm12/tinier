package settings

import (
	"fmt"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Image struct {
	// Extensions is the list of image file extensions to convert
	// from the input directory. Images with file extensions not
	// listed are simply copied to the output directory.
	Extensions []string
	// OutputExtension is the output extension to set on converted
	// image files. If defaults to `.avif`.
	OutputExtension string
	Scale           string
	// Codec is the codec to use, which defaults to `mjpeg`.
	Codec string
	// QScale is the constant quantizer to use, which defaults to 5.
	// Note this is only used for the `mjpeg` codec.
	QScale int
	// CRF is the constant quality to use, which defaults to 35.
	// Note this is only used for the `libaom-av1` codec.
	// See https://trac.ffmpeg.org/wiki/Encode/AV1#ConstantQuality
	CRF  int
	Skip *bool
}

func (i *Image) setDefaults() {
	i.Extensions = gosettings.DefaultSlice(i.Extensions, []string{".jpg", ".jpeg", ".png"})
	i.OutputExtension = gosettings.DefaultString(i.OutputExtension, ".jpg")
	i.Scale = gosettings.DefaultString(i.Scale, "1280:-1")
	i.Codec = gosettings.DefaultString(i.Codec, "mjpeg")
	const defaultQScale = 5
	i.QScale = gosettings.DefaultNumber(i.QScale, defaultQScale)
	const defaultCRF = 35
	i.CRF = gosettings.DefaultNumber(i.CRF, defaultCRF)
	i.Skip = gosettings.DefaultPointer(i.Skip, false)
}

func (i *Image) mergeWith(other Image) {
	i.Extensions = gosettings.MergeWithSlice(i.Extensions, other.Extensions)
	i.OutputExtension = gosettings.MergeWithString(i.OutputExtension, other.OutputExtension)
	i.Scale = gosettings.MergeWithString(i.Scale, other.Scale)
	i.Codec = gosettings.MergeWithString(i.Codec, other.Codec)
	i.QScale = gosettings.MergeWithNumber(i.QScale, other.QScale)
	i.CRF = gosettings.MergeWithNumber(i.CRF, other.CRF)
	i.Skip = gosettings.MergeWithPointer(i.Skip, other.Skip)
}

func (i *Image) overrideWith(other Image) {
	i.Extensions = gosettings.OverrideWithSlice(i.Extensions, other.Extensions)
	i.OutputExtension = gosettings.OverrideWithString(i.OutputExtension, other.OutputExtension)
	i.Scale = gosettings.OverrideWithString(i.Scale, other.Scale)
	i.Codec = gosettings.OverrideWithString(i.Codec, other.Codec)
	i.QScale = gosettings.OverrideWithNumber(i.QScale, other.QScale)
	i.CRF = gosettings.OverrideWithNumber(i.CRF, other.CRF)
	i.Skip = gosettings.OverrideWithPointer(i.Skip, other.Skip)
}

func (i *Image) validate() (err error) {
	err = validate.AllMatchRegex(i.Extensions, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed image file extension: %w", err)
	}

	err = validate.MatchRegex(i.OutputExtension, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed image output extension: %w", err)
	}

	err = validate.MatchRegex(i.Scale, regexScale)
	if err != nil {
		return fmt.Errorf("malformed image scale: %w", err)
	}

	err = validate.IsOneOf(i.Codec, "mjpeg", "libaom-av1")
	if err != nil {
		return fmt.Errorf("codec: %w", err)
	}

	const minQScale, maxQScale = 1, 31
	err = validate.NumberBetween(i.QScale, minQScale, maxQScale)
	if err != nil {
		return fmt.Errorf("image qscale value: %w", err)
	}

	return nil
}

func (i *Image) toLinesNode() *gotree.Node {
	if *i.Skip {
		return gotree.New("Image files: skip")
	}

	node := gotree.New("Image files:")
	node.Appendf("Input file extensions: %s", andStrings(i.Extensions))
	node.Appendf("Output file extension: %s", i.OutputExtension)
	node.Appendf("Scale: %s", i.Scale)
	switch i.Codec {
	case "mjpeg":
		codecNode := node.Appendf("Codec: %s", i.Codec)
		codecNode.Appendf("Constant quantizer qscale: %d", i.QScale)
	case "libaom-av1":
		codecNode := node.Appendf("Codec: libaom-av1")
		codecNode.Appendf("Constant quality CRF: %d", i.CRF)
	}
	return node
}

func (i *Image) String() string {
	return i.toLinesNode().String()
}
