package settings

import (
	"fmt"

	"github.com/qdm12/gosettings/defaults"
	"github.com/qdm12/gosettings/merge"
	"github.com/qdm12/gosettings/override"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Audio struct {
	// Extensions is the list of audio file extensions to convert
	// from the input directory. Audio files with file extensions
	// not listed are simply copied to the output directory.
	Extensions []string
	// OutputExtension is the output extension to set on converted
	// audio files. If defaults to `.mp3`.
	OutputExtension string
	QScale          *int
	Codec           string
	Skip            *bool
}

func (a *Audio) setDefaults() {
	a.Extensions = defaults.StringSlice(a.Extensions, []string{".mp3", ".flac"})
	a.OutputExtension = defaults.String(a.OutputExtension, ".mp3")
	const defaultQScale = 5
	a.QScale = defaults.IntPtr(a.QScale, defaultQScale)
	a.Codec = defaults.String(a.Codec, "libmp3lame")
	a.Skip = defaults.Bool(a.Skip, false)
}

func (a *Audio) mergeWith(other Audio) {
	a.Extensions = merge.StringSlices(a.Extensions, other.Extensions)
	a.OutputExtension = merge.String(a.OutputExtension, other.OutputExtension)
	a.QScale = merge.IntPtr(a.QScale, other.QScale)
	a.Codec = merge.String(a.Codec, other.Codec)
	a.Skip = merge.Bool(a.Skip, other.Skip)
}

func (a *Audio) overrideWith(other Audio) {
	a.Extensions = override.StringSlice(a.Extensions, other.Extensions)
	a.OutputExtension = override.String(a.OutputExtension, other.OutputExtension)
	a.QScale = override.IntPtr(a.QScale, other.QScale)
	a.Codec = override.String(a.Codec, other.Codec)
	a.Skip = override.Bool(a.Skip, other.Skip)
}

func (a *Audio) validate() (err error) {
	err = validate.AllMatchRegex(a.Extensions, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed audio file extension: %w", err)
	}

	err = validate.MatchRegex(a.OutputExtension, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed audio output extension: %w", err)
	}

	const minQScale, maxQScale = 0, 9
	err = validate.IntBetween(*a.QScale, minQScale, maxQScale)
	if err != nil {
		return fmt.Errorf("audio quality scale: %w", err)
	}

	return nil
}

func (a *Audio) toLinesNode() *gotree.Node {
	if *a.Skip {
		return gotree.New("Audio files: skip")
	}

	node := gotree.New("Audio files:")
	node.Appendf("Input file extensions: %s", andStrings(a.Extensions))
	node.Appendf("Output file extension: %s", a.OutputExtension)
	node.Appendf("Constant quantizer qscale: %d", *a.QScale)
	node.Appendf("Codec: %s", a.Codec)
	return node
}

func (a *Audio) String() string {
	return a.toLinesNode().String()
}
