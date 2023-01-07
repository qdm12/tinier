package flags

import (
	"flag"
	"fmt"

	"github.com/qdm12/tinier/internal/config/settings"
)

func (source *Source) Read() (settings settings.Settings, err error) {
	flagSet, flagSettings, rawStrings := configureFlagSet(source.args[0])
	if err := flagSet.Parse(source.args[1:]); err != nil {
		return settings, err
	}

	err = postProcessRawStrings(rawStrings, source.validator, flagSettings)
	if err != nil {
		return settings, err
	}

	flagSet.Visit(func(f *flag.Flag) {
		visitFlag(f.Name, &settings, *flagSettings)
	})

	return settings, nil
}

type rawStrings struct {
	videoExtensionsCSV string
	imageExtensionsCSV string
	audioExtensionsCSV string
}

func configureFlagSet(flagSetName string) (flagSet *flag.FlagSet,
	flagSettings *settings.Settings, rawStrings rawStrings) {
	flagSettings = new(settings.Settings)
	flagSet = flag.NewFlagSet(flagSetName, flag.ExitOnError)

	// set pointers to non-nil values and
	// use default values for flag documentation
	flagSettings.SetDefaults()

	// note the default values here are only for information purposes,
	// the actual default are set in the settings package.
	flagSet.StringVar(&flagSettings.InputDirPath, "inputdirpath", flagSettings.InputDirPath, "Input directory path.")
	flagSet.StringVar(&flagSettings.OutputDirPath, "outputdirpath", flagSettings.OutputDirPath, "Output directory path.")
	flagSet.StringVar(flagSettings.FfmpegPath, "ffmpegpath", *flagSettings.FfmpegPath, "FFMPEG binary path.")
	flagSet.StringVar(&flagSettings.FfmpegMinVersion, "ffmpegminversion",
		flagSettings.FfmpegMinVersion, "FFMPEG binary minimum version requirement.")
	flagSet.BoolVar(flagSettings.OverrideOutput, "override",
		*flagSettings.OverrideOutput, "Override files in the output directory.")

	configureFlagSetVideo(flagSet, flagSettings, &rawStrings.videoExtensionsCSV)
	configureFlagSetImage(flagSet, flagSettings, &rawStrings.imageExtensionsCSV)
	configureFlagSetAudio(flagSet, flagSettings, &rawStrings.audioExtensionsCSV)

	return flagSet, flagSettings, rawStrings
}

func postProcessRawStrings(rawStrings rawStrings, validator Validator,
	settings *settings.Settings) (err error) {
	err = postProcessVideo(&settings.Video, validator, rawStrings.videoExtensionsCSV)
	if err != nil {
		return err
	}

	err = postProcessImage(&settings.Image, validator, rawStrings.imageExtensionsCSV)
	if err != nil {
		return err
	}

	err = postProcessAudio(&settings.Audio, validator, rawStrings.audioExtensionsCSV)
	if err != nil {
		return err
	}

	return nil
}

type visiterFunc func(flagName string, destination *settings.Settings,
	source settings.Settings) (match bool)

func visitFlag(flagName string, destination *settings.Settings,
	source settings.Settings) {
	switch flagName {
	case "inputdirpath":
		destination.InputDirPath = source.InputDirPath
		return
	case "outputdirpath":
		destination.OutputDirPath = source.OutputDirPath
		return
	case "ffmpegpath":
		destination.FfmpegPath = source.FfmpegPath
		return
	case "ffmpegminversion":
		destination.FfmpegMinVersion = source.FfmpegMinVersion
		return
	case "override":
		destination.OverrideOutput = source.OverrideOutput
		return
	}

	for _, visiter := range [...]visiterFunc{
		visitVideoFlag,
		visitImageFlag,
		visitAudioFlag,
	} {
		if visiter(flagName, destination, source) {
			return
		}
	}

	panic(fmt.Sprintf("flag not added to switch case: %s", flagName))
}
