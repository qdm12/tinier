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

func New(cmd Runner, binPath string,
	minVersion semver.Semver, logger Logger) *FFMPEG {
	return &FFMPEG{
		cmd:        cmd,
		binPath:    binPath,
		minVersion: minVersion,
		logger:     logger,
	}
}
