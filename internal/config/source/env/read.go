package env

import (
	"fmt"

	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) Read() (settings settings.Settings, err error) {
	settings.InputDirPath = s.env.String("TINIER_INPUT_DIR_PATH")
	settings.OutputDirPath = s.env.String("TINIER_OUTPUT_DIR_PATH")
	settings.FfmpegPath = s.env.Get("TINIER_FFMPEG_PATH")
	settings.FfmpegMinVersion = s.env.String("TINIER_FFMPEG_MIN_VERSION")

	settings.OverrideOutput, err = s.env.BoolPtr("TINIER_OVERRIDE_OUTPUT")
	if err != nil {
		return settings, err
	}

	settings.Image, err = s.readImage()
	if err != nil {
		return settings, fmt.Errorf("image settings: %w", err)
	}

	settings.Video, err = s.readVideo()
	if err != nil {
		return settings, fmt.Errorf("video settings: %w", err)
	}

	settings.Audio, err = s.readAudio()
	if err != nil {
		return settings, fmt.Errorf("audio settings: %w", err)
	}

	settings.Log, err = s.readLog()
	if err != nil {
		return settings, fmt.Errorf("log settings: %w", err)
	}

	return settings, nil
}
