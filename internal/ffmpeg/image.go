package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var ErrConversion = errors.New("failed FFMPEG conversion")

func (f *FFMPEG) TinyImage(ctx context.Context, inputPath, outputPath,
	scale string, qScale int) (err error) {
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", inputPath,
		"-vf", "scale=" + scale,
		"-qscale:v", fmt.Sprint(qScale),
		"-map_metadata", "0",
		"-movflags", "use_metadata_tags",
	}

	args = append(args, outputPath, "-noautorotate")

	execCmd := exec.CommandContext(ctx, f.binPath, args...) //nolint:gosec
	patchCmd(execCmd)

	output, _ := f.cmd.Run(execCmd)
	if ctx.Err() != nil {
		return ctx.Err()
	} else if err != nil {
		return fmt.Errorf("%w: %s", ErrConversion, output)
	}
	return nil
}
