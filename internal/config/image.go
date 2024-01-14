package config

import (
	"fmt"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Image struct {
	// Extensions is the list of image file extensions to convert
	// from the input directory. Images with file extensions not
	// listed are simply copied to the output directory.
	Extensions []string
	// OutputExtension is the output extension to set on converted
	// image files. If defaults to `.jpg`.
	OutputExtension string
	Scale           string
	// Codec is the codec to use, which defaults to `mjpeg`.
	Codec string
	// QScale is the constant quantizer to use, which defaults to 5.
	// Note this is only used for the `mjpeg` codec.
	QScale uint
	// CRF is the constant quality to use, which defaults to 35.
	// Note this is only used for the `libaom-av1` codec.
	// See https://trac.ffmpeg.org/wiki/Encode/AV1#ConstantQuality
	CRF  uint
	Skip *bool
}

func (i *Image) setDefaults() {
	i.Extensions = gosettings.DefaultSlice(i.Extensions, []string{".jpg", ".jpeg", ".png", ".avif"})
	i.OutputExtension = gosettings.DefaultComparable(i.OutputExtension, ".jpg")
	i.Scale = gosettings.DefaultComparable(i.Scale, "1280:-1")
	i.Codec = gosettings.DefaultComparable(i.Codec, "mjpeg")
	const defaultQScale = 5
	i.QScale = gosettings.DefaultComparable(i.QScale, defaultQScale)
	const defaultCRF = 35
	i.CRF = gosettings.DefaultComparable(i.CRF, defaultCRF)
	i.Skip = gosettings.DefaultPointer(i.Skip, false)
}

func (i *Image) overrideWith(other Image) {
	i.Extensions = gosettings.OverrideWithSlice(i.Extensions, other.Extensions)
	i.OutputExtension = gosettings.OverrideWithComparable(i.OutputExtension, other.OutputExtension)
	i.Scale = gosettings.OverrideWithComparable(i.Scale, other.Scale)
	i.Codec = gosettings.OverrideWithComparable(i.Codec, other.Codec)
	i.QScale = gosettings.OverrideWithComparable(i.QScale, other.QScale)
	i.CRF = gosettings.OverrideWithComparable(i.CRF, other.CRF)
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

func (i *Image) read(reader *reader.Reader) (err error) {
	i.Scale = reader.String("IMAGE_SCALE")
	i.OutputExtension = reader.String("IMAGE_OUTPUT_EXTENSION")
	i.Extensions = reader.CSV("IMAGE_EXTENSIONS")

	i.Skip, err = reader.BoolPtr("IMAGE_SKIP")
	if err != nil {
		return err
	}

	i.Codec = reader.String("IMAGE_CODEC")
	i.CRF, err = reader.Uint("IMAGE_CRF")
	if err != nil {
		return err
	}

	i.QScale, err = reader.Uint("IMAGE_QSCALE")
	if err != nil {
		return err
	}

	return nil
}
