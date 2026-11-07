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
	// Scale is the scale of the video to write, in the scale syntax of
	// FFMPEG. It defaults to a width of at most 1280 pixels, which lets
	// a smaller video to its own width instead of upscaling it.
	Scale string
	// Preset is the preset of the video encoder, whose values are
	// encoder specific. It is left empty by default, and the preset of
	// the encoder is then chosen once the encoder of the conversion is
	// known: a codec this FFMPEG does not provide is replaced by
	// another one. The preset is 8 for SVT-AV1, and fast for the x264
	// and x265 encoders.
	Preset     string
	Codec      string
	AudioCodec string
	Crf        *uint
	Skip       *bool
}

func (v *Video) setDefaults() {
	v.Extensions = gosettings.DefaultSlice(v.Extensions, []string{".mp4", ".mov", ".avi"})
	v.OutputExtension = gosettings.DefaultComparable(v.OutputExtension, ".mp4")
	v.Codec = gosettings.DefaultComparable(v.Codec, defaultVideoCodec)
	v.Scale = gosettings.DefaultComparable(v.Scale, defaultVideoScale)
	v.AudioCodec = gosettings.DefaultComparable(v.AudioCodec, "libopus")
	const defaultCRF = 23
	v.Crf = gosettings.DefaultPointer(v.Crf, defaultCRF)
	v.Skip = gosettings.DefaultPointer(v.Skip, false)
}

// Names of the video encoders tinier knows about, which are the names FFMPEG
// gives them. Only the AV1 encoder of libaom and the two encoders of x264 and
// x265, held by the same library, take words as presets.
const (
	defaultVideoCodec = "libsvtav1"
	h264Codec         = "libx264"
	h265Codec         = "libx265"
	aomAV1Codec       = "libaom-av1"
	// defaultVideoScale caps the width at 1280 pixels without ever
	// reaching above the width of the input file, and asks FFMPEG for
	// an even height, which the crop that follows also guarantees.
	defaultVideoScale = "min(1280,iw):-2"
)

func (v *Video) overrideWith(other Video) {
	v.Extensions = gosettings.OverrideWithSlice(v.Extensions, other.Extensions)
	v.OutputExtension = gosettings.OverrideWithComparable(v.OutputExtension, other.OutputExtension)
	v.Scale = gosettings.OverrideWithComparable(v.Scale, other.Scale)
	v.Preset = gosettings.OverrideWithComparable(v.Preset, other.Preset)
	v.Codec = gosettings.OverrideWithComparable(v.Codec, other.Codec)
	v.AudioCodec = gosettings.OverrideWithComparable(v.AudioCodec, other.AudioCodec)
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

	err = validatePreset(v.Codec, v.Preset)
	if err != nil {
		return err
	}

	const minCRF, maxCRF = 0, 51
	err = validate.NumberBetween(*v.Crf, minCRF, maxCRF)
	if err != nil {
		return fmt.Errorf("video CRF: %w", err)
	}

	return nil
}

// DefaultPreset returns the preset to encode a video with when none was given.
// Preset values are encoder specific: SVT-AV1 takes numbers where the x264 and
// x265 encoders take words, and a preset of one makes FFMPEG refuse to open the
// encoder of the others. The preset can therefore only be chosen once the
// encoder of the conversion is known, which is after the build of FFMPEG has
// been read: a codec it does not provide is replaced by another encoder.
func (v *Video) DefaultPreset() string {
	switch strings.ToLower(v.Codec) {
	case h264Codec, h265Codec, aomAV1Codec:
		return "fast"
	default:
		// The preset of SVT-AV1, the encoder tinier defaults to, is a number the
		// encoder itself defaults to.
		return "8"
	}
}

// validatePreset tells if the given preset is one of the presets of the given
// video encoder. An empty preset is valid: it asks for the preset of the
// encoder, which DefaultPreset returns.
func validatePreset(codec, preset string) error {
	if preset == "" {
		// No preset was given: the preset of the encoder chosen for the conversion
		// is used, and that one is a preset of that encoder by construction.
		return nil
	}

	var validPresets []string
	switch strings.ToLower(codec) {
	case defaultVideoCodec:
		validPresets = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	case h264Codec, h265Codec, aomAV1Codec:
		validPresets = []string{"ultrafast", "superfast", "veryfast", "faster", "fast",
			"medium", "slow", "slower", "veryslow", "placebo"}
	}
	if len(validPresets) == 0 {
		return nil
	}

	err := validate.IsOneOf(preset, validPresets...)
	if err != nil {
		return fmt.Errorf("preset is unknown for codec %s: %w", codec, err)
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
	node.Appendf("Preset: %s", presetLabel(v.Preset))
	node.Appendf("Codec: %s", v.Codec)
	node.Appendf("Audio codec: %s", v.AudioCodec)
	node.Appendf("Constant rate factor: %d", *v.Crf)
	return node
}

// presetLabel returns what to print for the given preset, whose absence means
// the preset of the encoder of the conversion is used.
func presetLabel(preset string) string {
	if preset == "" {
		return "none, the preset of the encoder"
	}

	return preset
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
	v.AudioCodec = reader.String("VIDEO_AUDIO_CODEC")

	v.Crf, err = reader.UintPtr("VIDEO_CRF")
	if err != nil {
		return err
	}

	v.Skip, err = reader.BoolPtr("VIDEO_SKIP")
	if err != nil {
		return err
	}

	return nil
}
