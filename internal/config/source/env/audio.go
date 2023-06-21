package env

import (
	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) readAudio() (settings settings.Audio, err error) {
	settings.Codec = s.env.String("TINIER_AUDIO_CODEC")
	settings.OutputExtension = s.env.String("TINIER_AUDIO_OUTPUT_EXTENSION")

	settings.Extensions = s.env.CSV("TINIER_AUDIO_EXTENSIONS")

	settings.Skip, err = s.env.BoolPtr("TINIER_AUDIO_SKIP")
	if err != nil {
		return settings, err
	}

	settings.QScale, err = s.env.IntPtr("TINIER_AUDIO_QSCALE")
	if err != nil {
		return settings, err
	}

	settings.BitRate = s.env.Get("TINIER_AUDIO_BITRATE")

	return settings, nil
}
