package flags

import (
	"flag"
	"fmt"

	"github.com/qdm12/log"
	"github.com/qdm12/tinier/internal/config/settings"
)

func configureFlagSetLog(flagSet *flag.FlagSet, logLevel *string) {
	flagSet.StringVar(logLevel, "loglevel", "info", "Logging level")
}

func postProcessLog(settings *settings.Log, logLevel string) (err error) {
	if logLevel != "" {
		settings.Level, err = log.ParseLevel(logLevel)
		if err != nil {
			return fmt.Errorf("flag -loglevel: %w", err)
		}
	}

	return nil
}

func visitLogFlag(flagName string, destination *settings.Settings,
	source settings.Settings) (match bool) {
	switch flagName { //nolint:gocritic
	case "loglevel":
		destination.Log.Level = source.Log.Level
		return true
	}
	return false
}
