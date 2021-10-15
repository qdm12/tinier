package flags

import (
	"flag"
	"fmt"
	"strings"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func configureFlagSetVideo(flagSet *flag.FlagSet, flagSettings *settings.Settings,
	extensionsCSV *string) {
	flagSet.BoolVar(flagSettings.Video.Skip, "videoskip", *flagSettings.Video.Skip, "Skip video files.")
	flagSet.StringVar(extensionsCSV, "videoextensions",
		strings.Join(flagSettings.Video.Extensions, ","), "CSV list of video file extensions.")
	flagSet.StringVar(&flagSettings.Video.OutputExtension, "videooutputextension",
		flagSettings.Video.OutputExtension, "Video output file extension to use.")
	flagSet.StringVar(&flagSettings.Video.Scale, "videoscale", flagSettings.Video.Scale, "Video ffmpeg scale value.")
	flagSet.StringVar(&flagSettings.Video.Preset, "videopreset", flagSettings.Video.Preset, "Video ffmpeg preset.")
	flagSet.StringVar(&flagSettings.Video.Codec, "videocodec", flagSettings.Video.Codec, "Video ffmpeg codec.")
	flagSet.IntVar(flagSettings.Video.Crf, "videocrf", *flagSettings.Video.Crf, "Video ffmpeg CRF value.")
}

func postProcessVideo(settings *settings.Video, validator Validator,
	extensionsCSV string) (err error) {
	if extensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(
			extensionsCSV, separated.OptionLowercase(),
			separated.OptionIgnoreEmpty())
		if err != nil {
			return fmt.Errorf("flag -videoextensions: %w", err)
		}
	}

	return nil
}

func visitVideoFlag(flagName string, destination *settings.Settings,
	source settings.Settings) (match bool) {
	switch flagName {
	case "videoskip":
		destination.Video.Skip = source.Video.Skip
		return true
	case "videoextensions":
		destination.Video.Extensions = source.Video.Extensions
		return true
	case "videooutputextension":
		destination.Video.OutputExtension = source.Video.OutputExtension
		return true
	case "videoscale":
		destination.Video.Scale = source.Video.Scale
		return true
	case "videopreset":
		destination.Video.Preset = source.Video.Preset
		return true
	case "videocodec":
		destination.Video.Codec = source.Video.Codec
		return true
	case "videocrf":
		destination.Video.Crf = source.Video.Crf
		return true
	}
	return false
}
