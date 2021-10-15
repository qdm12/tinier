package env

import (
	"fmt"
	"os"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readVideo(validator Validator) (settings settings.Video, err error) {
	settings.Scale = os.Getenv("TINIER_VIDEO_SCALE")
	settings.Preset = os.Getenv("TINIER_VIDEO_PRESET")
	settings.Codec = os.Getenv("TINIER_VIDEO_CODEC")
	settings.OutputExtension = os.Getenv("TINIER_VIDEO_OUTPUT_EXTENSION")

	videoExtensionsCSV := os.Getenv("TINIER_VIDEO_EXTENSIONS")
	if videoExtensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(videoExtensionsCSV,
			separated.OptionLowercase(), separated.OptionIgnoreEmpty())
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_VIDEO_EXTENSIONS: %w", err)
		}
	}

	skipVideoStr := os.Getenv("TINIER_VIDEO_SKIP")
	if skipVideoStr != "" {
		settings.Skip = new(bool)
		*settings.Skip, err = validator.ValidateBinary(skipVideoStr)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_VIDEO_SKIP: %w", err)
		}
	}

	videoCRF := os.Getenv("TINIER_VIDEO_CRF")
	if videoCRF != "" {
		crf, err := validator.ValidateInteger(videoCRF)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_VIDEO_CRF: %w", err)
		}
		settings.Crf = &crf
	}

	return settings, nil
}
