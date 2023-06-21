package env

import (
	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) readImage() (settings settings.Image, err error) {
	settings.Scale = s.env.String("TINIER_IMAGE_SCALE")
	settings.OutputExtension = s.env.String("TINIER_IMAGE_OUTPUT_EXTENSION")
	settings.Extensions = s.env.CSV("TINIER_IMAGE_EXTENSIONS")

	settings.Skip, err = s.env.BoolPtr("TINIER_IMAGE_SKIP")
	if err != nil {
		return settings, err
	}

	settings.Codec = s.env.String("TINIER_IMAGE_CODEC")
	settings.CRF, err = s.env.Int("TINIER_IMAGE_CRF")
	if err != nil {
		return settings, err
	}

	settings.QScale, err = s.env.Int("TINIER_IMAGE_QSCALE")
	if err != nil {
		return settings, err
	}

	return settings, nil
}
