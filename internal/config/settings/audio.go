package settings

import (
	"errors"
	"fmt"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Audio struct {
	// Extensions is the list of audio file extensions to convert
	// from the input directory. Audio files with file extensions
	// not listed are simply copied to the output directory.
	Extensions []string
	// OutputExtension is the output extension to set on converted
	// audio files. If defaults to `.opus`.
	OutputExtension string
	QScale          *int
	Codec           string
	// BitRate is the bitrate string to use for the codec.
	// It defaults to 32k if the libopus codec is used.
	// It can be set to the empty string so the qscale parameter is used
	// instead of the bitrate.
	BitRate *string
	Skip    *bool
}

func (a *Audio) setDefaults() {
	a.Extensions = gosettings.DefaultSlice(a.Extensions, []string{".mp3", ".flac"})
	a.OutputExtension = gosettings.DefaultString(a.OutputExtension, ".opus")
	const defaultQScale = 5
	a.QScale = gosettings.DefaultPointer(a.QScale, defaultQScale)
	a.Codec = gosettings.DefaultString(a.Codec, "libopus")
	if a.Codec == "libopus" { // bit rate is required for libopus
		a.BitRate = gosettings.DefaultPointer(a.BitRate, "32k")
	} else { // default to empty string to signal not to use it.
		a.BitRate = gosettings.DefaultPointer(a.BitRate, "")
	}
	a.Skip = gosettings.DefaultPointer(a.Skip, false)
}

func (a *Audio) mergeWith(other Audio) {
	a.Extensions = gosettings.MergeWithSlice(a.Extensions, other.Extensions)
	a.OutputExtension = gosettings.MergeWithString(a.OutputExtension, other.OutputExtension)
	a.QScale = gosettings.MergeWithPointer(a.QScale, other.QScale)
	a.Codec = gosettings.MergeWithString(a.Codec, other.Codec)
	a.BitRate = gosettings.MergeWithPointer(a.BitRate, other.BitRate)
	a.Skip = gosettings.MergeWithPointer(a.Skip, other.Skip)
}

func (a *Audio) overrideWith(other Audio) {
	a.Extensions = gosettings.OverrideWithSlice(a.Extensions, other.Extensions)
	a.OutputExtension = gosettings.OverrideWithString(a.OutputExtension, other.OutputExtension)
	a.QScale = gosettings.OverrideWithPointer(a.QScale, other.QScale)
	a.Codec = gosettings.OverrideWithString(a.Codec, other.Codec)
	a.BitRate = gosettings.OverrideWithPointer(a.BitRate, other.BitRate)
	a.Skip = gosettings.OverrideWithPointer(a.Skip, other.Skip)
}

var ErrBitRateNotSet = errors.New("bit rate is not set")

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
	err = validate.NumberBetween(*a.QScale, minQScale, maxQScale)
	if err != nil {
		return fmt.Errorf("audio quality scale: %w", err)
	}

	if a.Codec == "libopus" && *a.BitRate == "" { // bit rate is required for libopus
		return fmt.Errorf("%w: for audio codec %s", ErrBitRateNotSet, a.Codec)
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
	if *a.BitRate != "" {
		node.Appendf("Bitrate: %s", *a.BitRate)
	}
	return node
}

func (a *Audio) String() string {
	return a.toLinesNode().String()
}
