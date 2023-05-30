package env

import (
	"fmt"

	"github.com/qdm12/gosettings/sources/env"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readImage() (settings settings.Image, err error) {
	settings.Scale = env.Get("TINIER_IMAGE_SCALE")
	settings.OutputExtension = env.Get("TINIER_IMAGE_OUTPUT_EXTENSION")
	settings.Extensions = env.CSV("TINIER_IMAGE_EXTENSIONS")

	settings.Skip, err = env.BoolPtr("TINIER_IMAGE_SKIP")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_IMAGE_SKIP: %w", err)
	}

	settings.QScale, err = env.Int("TINIER_IMAGE_QSCALE")
	if err != nil {
		return settings, fmt.Errorf("environment variable TINIER_IMAGE_QSCALE: %w", err)
	}

	return settings, nil
}
