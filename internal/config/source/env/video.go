package env

import (
	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) readVideo() (settings settings.Video, err error) {
	settings.Scale = s.env.String("TINIER_VIDEO_SCALE")
	settings.Preset = s.env.String("TINIER_VIDEO_PRESET")
	settings.Codec = s.env.String("TINIER_VIDEO_CODEC")
	settings.OutputExtension = s.env.String("TINIER_VIDEO_OUTPUT_EXTENSION")

	settings.Extensions = s.env.CSV("TINIER_VIDEO_EXTENSIONS")

	settings.Skip, err = s.env.BoolPtr("TINIER_VIDEO_SKIP")
	if err != nil {
		return settings, err
	}

	settings.Crf, err = s.env.IntPtr("TINIER_VIDEO_CRF")
	if err != nil {
		return settings, err
	}

	return settings, nil
}
