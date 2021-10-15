package flags

import (
	"flag"
	"fmt"
	"strings"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func configureFlagSetAudio(flagSet *flag.FlagSet, flagSettings *settings.Settings,
	extensionsCSV *string) {
	flagSet.BoolVar(flagSettings.Audio.Skip, "audioskip", *flagSettings.Audio.Skip, "Skip audio files.")
	flagSet.StringVar(extensionsCSV, "audioextensions",
		strings.Join(flagSettings.Audio.Extensions, ","), "CSV list of audio file extensions.")
	flagSet.StringVar(&flagSettings.Audio.OutputExtension, "audiooutputextension",
		flagSettings.Audio.OutputExtension, "Audio output file extension to use.")
	flagSet.IntVar(flagSettings.Audio.QScale, "audioqscale", *flagSettings.Audio.QScale, "Audio ffmpeg QScale value.")
	flagSet.StringVar(&flagSettings.Audio.Codec, "audiocodec", flagSettings.Audio.Codec, "Audio ffmpeg codec.")
}

func postProcessAudio(settings *settings.Audio, validator Validator,
	extensionsCSV string) (err error) {
	if extensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(
			extensionsCSV, separated.OptionLowercase(),
			separated.OptionIgnoreEmpty())
		if err != nil {
			return fmt.Errorf("flag -audioextensions: %w", err)
		}
	}

	return nil
}

func visitAudioFlag(flagName string, destination *settings.Settings,
	source settings.Settings) (match bool) {
	switch flagName {
	case "audioskip":
		destination.Audio.Skip = source.Audio.Skip
		return true
	case "audioextensions":
		destination.Audio.Extensions = source.Audio.Extensions
		return true
	case "audiooutputextension":
		destination.Audio.OutputExtension = source.Audio.OutputExtension
		return true
	case "audioqscale":
		destination.Audio.QScale = source.Audio.QScale
		return true
	case "audiocodec":
		destination.Audio.Codec = source.Audio.Codec
		return true
	}
	return false
}
