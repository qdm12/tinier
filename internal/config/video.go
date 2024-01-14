package config

import (
	"fmt"
	"strings"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
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
	v.Extensions = gosettings.DefaultSlice(v.Extensions, []string{".mp4", ".mov", ".avi"})
	v.OutputExtension = gosettings.DefaultComparable(v.OutputExtension, ".mp4")
	v.Scale = gosettings.DefaultComparable(v.Scale, "1280:-1")
	v.Preset = gosettings.DefaultComparable(v.Preset, "8")
	v.Codec = gosettings.DefaultComparable(v.Codec, "libsvtav1")
	const defaultCRF = 23
	v.Crf = gosettings.DefaultPointer(v.Crf, defaultCRF)
	v.Skip = gosettings.DefaultPointer(v.Skip, false)
}

func (v *Video) overrideWith(other Video) {
	v.Extensions = gosettings.OverrideWithSlice(v.Extensions, other.Extensions)
	v.OutputExtension = gosettings.OverrideWithComparable(v.OutputExtension, other.OutputExtension)
	v.Scale = gosettings.OverrideWithComparable(v.Scale, other.Scale)
	v.Preset = gosettings.OverrideWithComparable(v.Preset, other.Preset)
	v.Codec = gosettings.OverrideWithComparable(v.Codec, other.Codec)
	v.Crf = gosettings.OverrideWithPointer(v.Crf, other.Crf)
	v.Skip = gosettings.OverrideWithPointer(v.Skip, other.Skip)
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
	err = validate.NumberBetween(*v.Crf, minCRF, maxCRF)
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

func (v *Video) read(reader *reader.Reader) (err error) {
	v.Extensions = reader.CSV("VIDEO_EXTENSIONS")
	v.OutputExtension = reader.String("VIDEO_OUTPUT_EXTENSION")
	v.Scale = reader.String("VIDEO_SCALE")
	v.Preset = reader.String("VIDEO_PRESET")
	v.Codec = reader.String("VIDEO_CODEC")

	v.Crf, err = reader.IntPtr("VIDEO_CRF")
	if err != nil {
		return err
	}

	v.Skip, err = reader.BoolPtr("VIDEO_SKIP")
	if err != nil {
		return err
	}

	return nil
}
