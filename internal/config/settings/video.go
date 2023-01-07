package settings

import (
	"fmt"
	"strings"

	"github.com/qdm12/gosettings/defaults"
	"github.com/qdm12/gosettings/merge"
	"github.com/qdm12/gosettings/override"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Video struct {
	// Extensions is the list of video file extensions to convert
	// from the input directory. Videos with file extensions not
	// listed are simply copied to the output directory.
	Extensions []string
	// OutputExtension is the output extension to set on converted
	// video files. If defaults to `.mp4`.
	OutputExtension string
	Scale           string
	Preset          string
	Codec           string
	Crf             *int
	Skip            *bool
}

func (v *Video) setDefaults() {
	v.Extensions = defaults.StringSlice(v.Extensions, []string{".mp4", ".mov", ".avi"})
	v.OutputExtension = defaults.String(v.OutputExtension, ".mp4")
	v.Scale = defaults.String(v.Scale, "1280:-1")
	v.Preset = defaults.String(v.Preset, "8")
	v.Codec = defaults.String(v.Codec, "libsvtav1")
	const defaultCRF = 23
	v.Crf = defaults.IntPtr(v.Crf, defaultCRF)
	v.Skip = defaults.Bool(v.Skip, false)
}

func (v *Video) mergeWith(other Video) {
	v.Extensions = merge.StringSlices(v.Extensions, other.Extensions)
	v.OutputExtension = merge.String(v.OutputExtension, other.OutputExtension)
	v.Scale = merge.String(v.Scale, other.Scale)
	v.Preset = merge.String(v.Preset, other.Preset)
	v.Codec = merge.String(v.Codec, other.Codec)
	v.Crf = merge.IntPtr(v.Crf, other.Crf)
	v.Skip = merge.Bool(v.Skip, other.Skip)
}

func (v *Video) overrideWith(other Video) {
	v.Extensions = override.StringSlice(v.Extensions, other.Extensions)
	v.OutputExtension = override.String(v.OutputExtension, other.OutputExtension)
	v.Scale = override.String(v.Scale, other.Scale)
	v.Preset = override.String(v.Preset, other.Preset)
	v.Codec = override.String(v.Codec, other.Codec)
	v.Crf = override.IntPtr(v.Crf, other.Crf)
	v.Skip = override.Bool(v.Skip, other.Skip)
}

func (v *Video) validate() (err error) {
	err = validate.AllMatchRegex(v.Extensions, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed video file extension: %w", err)
	}

	err = validate.MatchRegex(v.OutputExtension, regexExtension)
	if err != nil {
		return fmt.Errorf("malformed video output extension: %w", err)
	}

	err = validate.MatchRegex(v.Scale, regexScale)
	if err != nil {
		return fmt.Errorf("malformed video scale: %w", err)
	}

	var validPresets []string
	switch strings.ToLower(v.Codec) {
	case "libsvtav1":
		validPresets = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	case "libx264", "libx265", "libaom-av1":
		validPresets = []string{"ultrafast", "superfast", "veryfast", "faster", "fast",
			"medium", "slow", "slower", "veryslow", "placebo"}
	}
	if len(validPresets) > 0 {
		err = validate.IsOneOf(v.Preset, validPresets...)
		if err != nil {
			return fmt.Errorf("preset is unknown for codec %s: %w", v.Codec, err)
		}
	}

	const minCRF, maxCRF = 0, 51
	err = validate.IntBetween(*v.Crf, minCRF, maxCRF)
	if err != nil {
		return fmt.Errorf("video CRF: %w", err)
	}

	return nil
}

func (v *Video) toLinesNode() *gotree.Node {
	if *v.Skip {
		return gotree.New("Video files: skip")
	}

	node := gotree.New("Video files:")
	node.Appendf("Input file extensions: %s", andStrings(v.Extensions))
	node.Appendf("Output file extension: %s", v.OutputExtension)
	node.Appendf("Scale: %s", v.Scale)
	node.Appendf("Preset: %s", v.Preset)
	node.Appendf("Codec: %s", v.Codec)
	node.Appendf("Constant rate factor: %d", *v.Crf)
	return node
}

func (v *Video) String() string {
	return v.toLinesNode().String()
}
