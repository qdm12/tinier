package env

import (
	"fmt"

	"github.com/qdm12/gosettings/sources/env"
	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) Read() (settings settings.Settings, err error) {
	settings.InputDirPath = env.Get("TINIER_INPUT_DIR_PATH")
	settings.OutputDirPath = env.Get("TINIER_OUTPUT_DIR_PATH")
	settings.FfmpegPath = env.StringPtr("TINIER_FFMPEG_PATH")
	settings.FfmpegMinVersion = env.Get("TINIER_FFMPEG_MIN_VERSION")

	settings.OverrideOutput, err = env.BoolPtr("TINIER_OVERRIDE_OUTPUT")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_OVERRIDE_OUTPUT: %w", err)
	}

	settings.Image, err = readImage()
	if err != nil {
		return settings, fmt.Errorf("image settings: %w", err)
	}

	settings.Video, err = readVideo()
	if err != nil {
		return settings, fmt.Errorf("video settings: %w", err)
	}

	settings.Audio, err = readAudio()
	if err != nil {
		return settings, fmt.Errorf("audio settings: %w", err)
	}

	return settings, nil
}
