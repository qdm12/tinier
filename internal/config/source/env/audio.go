package env

import (
	"fmt"
	"os"

	"github.com/qdm12/govalid"
	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readAudio() (settings settings.Audio, err error) {
	settings.Codec = os.Getenv("TINIER_AUDIO_CODEC")
	settings.OutputExtension = os.Getenv("TINIER_AUDIO_OUTPUT_EXTENSION")

	audioExtensionsCSV := os.Getenv("TINIER_AUDIO_EXTENSIONS")
	if audioExtensionsCSV != "" {
		settings.Extensions, err = govalid.ValidateSeparated(audioExtensionsCSV,
			separated.OptionLowercase(), separated.OptionIgnoreEmpty())
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_AUDIO_EXTENSIONS: %w", err)
		}
	}

	skipAudioStr := os.Getenv("TINIER_AUDIO_SKIP")
	if skipAudioStr != "" {
		settings.Skip, err = govalid.ValidateBinary(skipAudioStr)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_AUDIO_SKIP: %w", err)
		}
	}

	audioQScale := os.Getenv("TINIER_AUDIO_QSCALE")
	if audioQScale != "" {
		qscale, err := govalid.ValidateInteger(audioQScale)
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
