package env

import (
	"fmt"
	"os"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func readImage(validator Validator) (settings settings.Image, err error) {
	settings.Scale = os.Getenv("TINIER_IMAGE_SCALE")
	settings.OutputExtension = os.Getenv("TINIER_IMAGE_OUTPUT_EXTENSION")
	imageExtensionsCSV := os.Getenv("TINIER_IMAGE_EXTENSIONS")
	if imageExtensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(imageExtensionsCSV,
			separated.OptionLowercase(), separated.OptionIgnoreEmpty())
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_IMAGE_EXTENSIONS: %w", err)
		}
	}

	skipImageStr := os.Getenv("TINIER_IMAGE_SKIP")
	if skipImageStr != "" {
		settings.Skip = new(bool)
		*settings.Skip, err = validator.ValidateBinary(skipImageStr)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_IMAGE_SKIP: %w", err)
		}
	}

	imageQScale := os.Getenv("TINIER_IMAGE_QSCALE")
	if imageQScale != "" {
		settings.QScale, err = validator.ValidateInteger(imageQScale)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_IMAGE_QSCALE: %w", err)
		}
	}

	return settings, nil
}
