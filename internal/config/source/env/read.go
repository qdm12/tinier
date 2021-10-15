package env

import (
	"fmt"
	"os"

	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) Read() (settings settings.Settings, err error) {
	settings.InputDirPath = os.Getenv("TINIER_INPUT_DIR_PATH")
	settings.OutputDirPath = os.Getenv("TINIER_OUTPUT_DIR_PATH")
	ffmpegPath := os.Getenv("TINIER_FFMPEG_PATH")
	if ffmpegPath != "" {
		settings.FfmpegPath = &ffmpegPath
	}
	settings.FfmpegMinVersion = os.Getenv("TINIER_FFMPEG_MIN_VERSION")

	overrideOutputStr := os.Getenv("TINIER_OVERRIDE_OUTPUT")
	if overrideOutputStr != "" {
		settings.OverrideOutput = new(bool)
		*settings.OverrideOutput, err = s.validator.ValidateBinary(overrideOutputStr)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_OVERRIDE_OUTPUT: %w", err)
		}
	}

	settings.Image, err = readImage(s.validator)
	if err != nil {
		return settings, fmt.Errorf("image settings: %w", err)
	}

	settings.Video, err = readVideo(s.validator)
	if err != nil {
		return settings, fmt.Errorf("video settings: %w", err)
	}

	settings.Audio, err = readAudio(s.validator)
	if err != nil {
		return settings, fmt.Errorf("audio settings: %w", err)
	}

	return settings, nil
}
