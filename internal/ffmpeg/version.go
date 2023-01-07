package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/qdm12/tinier/internal/semver"
)

func (f *FFMPEG) Version(ctx context.Context) (version semver.Semver, err error) {
	return getVersion(ctx, f.binPath, f.cmd)
}

func getVersion(ctx context.Context, path string, runner Runner) (
	version semver.Semver, err error) {
	cmd := exec.CommandContext(ctx, path, "-version")
	patchCmd(cmd)
	s, err := runner.Run(cmd)
	if err != nil {
		return version, err
	}

	firstLine := strings.Split(s, "\n")[0]

	version, err = semver.Extract(firstLine)
	if err != nil {
		return version, fmt.Errorf("extracting semver: %w", err)
	}

	return version, nil
}
