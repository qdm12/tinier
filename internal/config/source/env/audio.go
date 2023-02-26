package env

import (
	"fmt"
	"os"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readAudio(validator Validator) (settings settings.Audio, err error) {
	settings.Codec = os.Getenv("TINIER_AUDIO_CODEC")
	settings.OutputExtension = os.Getenv("TINIER_AUDIO_OUTPUT_EXTENSION")

	audioExtensionsCSV := os.Getenv("TINIER_AUDIO_EXTENSIONS")
	if audioExtensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(audioExtensionsCSV,
			separated.OptionLowercase(), separated.OptionIgnoreEmpty())
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_AUDIO_EXTENSIONS: %w", err)
		}
	}

	skipAudioStr := os.Getenv("TINIER_AUDIO_SKIP")
	if skipAudioStr != "" {
		settings.Skip = new(bool)
		*settings.Skip, err = validator.ValidateBinary(skipAudioStr)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_AUDIO_SKIP: %w", err)
		}
	}

	audioQScale := os.Getenv("TINIER_AUDIO_QSCALE")
	if audioQScale != "" {
		qscale, err := validator.ValidateInteger(audioQScale)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_AUDIO_QSCALE: %w", err)
		}
		settings.QScale = &qscale
	}

	audioBitrate := os.Getenv("TINIER_AUDIO_BITRATE")
	if audioBitrate != "" {
		settings.BitRate = &audioBitrate
	}

	return settings, nil
}
