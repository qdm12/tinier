package env

import (
	"fmt"

	"github.com/qdm12/log"
	"github.com/qdm12/tinier/internal/config/settings"
)

func (s *Source) readLog() (settings settings.Log, err error) {
	level := s.env.String("TINIER_LOG_LEVEL")
	if level != "" {
		settings.Level, err = log.ParseLevel(level)
		if err != nil {
			return settings, fmt.Errorf("environment variable TINIER_LOG_LEVEL: %w", err)
		}
	}

	return settings, nil
}
