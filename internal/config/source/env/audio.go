package env

import (
	"fmt"

	"github.com/qdm12/gosettings/sources/env"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readAudio() (settings settings.Audio, err error) {
	settings.Codec = env.Get("TINIER_AUDIO_CODEC")
	settings.OutputExtension = env.Get("TINIER_AUDIO_OUTPUT_EXTENSION")

	settings.Extensions = env.CSV("TINIER_AUDIO_EXTENSIONS")

	settings.Skip, err = env.BoolPtr("TINIER_AUDIO_SKIP")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_AUDIO_SKIP: %w", err)
	}

	settings.QScale, err = env.IntPtr("TINIER_AUDIO_QSCALE")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_AUDIO_QSCALE: %w", err)
	}

	settings.BitRate = env.StringPtr("TINIER_AUDIO_BITRATE")

	return settings, nil
}
