package env

import (
	"fmt"

	"github.com/qdm12/gosettings/sources/env"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readVideo() (settings settings.Video, err error) {
	settings.Scale = env.Get("TINIER_VIDEO_SCALE")
	settings.Preset = env.Get("TINIER_VIDEO_PRESET")
	settings.Codec = env.Get("TINIER_VIDEO_CODEC")
	settings.OutputExtension = env.Get("TINIER_VIDEO_OUTPUT_EXTENSION")

	settings.Extensions = env.CSV("TINIER_VIDEO_EXTENSIONS")

	settings.Skip, err = env.BoolPtr("TINIER_VIDEO_SKIP")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_VIDEO_SKIP: %w", err)
	}

	settings.Crf, err = env.IntPtr("TINIER_VIDEO_CRF")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_VIDEO_CRF: %w", err)
	}

	return settings, nil
}
