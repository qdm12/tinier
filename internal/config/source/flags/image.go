package flags

import (
	"flag"
	"fmt"
	"strings"

	"github.com/qdm12/govalid/separated"
	"github.com/qdm12/tinier/internal/config/settings"
)

func configureFlagSetImage(flagSet *flag.FlagSet, flagSettings *settings.Settings,
	extensionsCSV *string) {
	flagSet.BoolVar(flagSettings.Image.Skip, "imageskip", *flagSettings.Image.Skip, "Skip image files.")
	flagSet.StringVar(extensionsCSV, "imageextensions",
		strings.Join(flagSettings.Image.Extensions, ","), "CSV list of image file extensions.")
	flagSet.StringVar(&flagSettings.Image.Scale, "imagescale", flagSettings.Image.Scale, "Image ffmpeg scale value.")
	flagSet.StringVar(&flagSettings.Image.OutputExtension, "imageoutputextension",
		flagSettings.Image.OutputExtension, "Image output file extension to use.")
	flagSet.IntVar(&flagSettings.Image.QScale, "imageqv", flagSettings.Image.QScale, "Image ffmpeg q:v value.")
}

func postProcessImage(settings *settings.Image, validator Validator,
	extensionsCSV string) (err error) {
	if extensionsCSV != "" {
		settings.Extensions, err = validator.ValidateSeparated(
			extensionsCSV, separated.OptionLowercase(),
			separated.OptionIgnoreEmpty())
		if err != nil {
			return fmt.Errorf("flag -imageextensions: %w", err)
		}
	}

	return nil
}

func visitImageFlag(flagName string, destination *settings.Settings,
	source settings.Settings) (match bool) {
	switch flagName {
	case "imageskip":
		destination.Image.Skip = source.Image.Skip
		return true
	case "imageextensions":
		destination.Image.Extensions = source.Image.Extensions
		return true
	case "imagescale":
		destination.Image.Scale = source.Image.Scale
		return true
	case "imageoutputextension":
		destination.Image.OutputExtension = source.Image.OutputExtension
		return true
	case "imageqv":
		destination.Image.QScale = source.Image.QScale
		return true
	}
	return false
}
