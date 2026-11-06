package ffmpeg

import (
	"github.com/qdm12/tinier/internal/semver"
)

type FFMPEG struct {
	cmd        Runner
	binPath    string
	minVersion semver.Semver
	logger     Logger
}

// baseArgs returns the arguments common to every FFMPEG command: overwrite the
// output file without asking, and log the warnings of the encoder only.
func baseArgs() []string {
	return []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
	}
}

func New(cmd Runner, binPath string,
	minVersion semver.Semver, logger Logger) *FFMPEG {
	return &FFMPEG{
		cmd:        cmd,
		binPath:    binPath,
		minVersion: minVersion,
		logger:     logger,
	}
}
