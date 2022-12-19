package settings

import (
	"fmt"

	"github.com/qdm12/gosettings/defaults"
	"github.com/qdm12/gosettings/merge"
	"github.com/qdm12/gosettings/override"
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
	QScale          int
	Skip            *bool
}

func (i *Image) setDefaults() {
	i.Extensions = defaults.StringSlice(i.Extensions, []string{".jpg", ".jpeg"})
	i.OutputExtension = defaults.String(i.OutputExtension, ".jpg")
	i.Scale = defaults.String(i.Scale, "1280:-1")
	const defaultQScale = 5
	i.QScale = defaults.Int(i.QScale, defaultQScale)
	i.Skip = defaults.Bool(i.Skip, false)
}

func (i *Image) mergeWith(other Image) {
	i.Extensions = merge.StringSlices(i.Extensions, other.Extensions)
	i.OutputExtension = merge.String(i.OutputExtension, other.OutputExtension)
	i.Scale = merge.String(i.Scale, other.Scale)
	i.QScale = merge.Int(i.QScale, other.QScale)
	i.Skip = merge.Bool(i.Skip, other.Skip)
}

func (i *Image) overrideWith(other Image) {
	i.Extensions = override.StringSlice(i.Extensions, other.Extensions)
	i.OutputExtension = override.String(i.OutputExtension, other.OutputExtension)
	i.Scale = override.String(i.Scale, other.Scale)
	i.QScale = override.Int(i.QScale, other.QScale)
	i.Skip = override.Bool(i.Skip, other.Skip)
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

	const minQScale, maxQScale = 1, 31
	err = validate.IntBetween(i.QScale, minQScale, maxQScale)
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
	node.Appendf("Constant quantizer qscale: %d", i.QScale)
	return node
}

func (i *Image) String() string {
	return i.toLinesNode().String()
}
