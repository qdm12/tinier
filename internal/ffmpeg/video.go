package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
)

func (f *FFMPEG) TinyVideo(ctx context.Context, inputPath, outputPath,
	scale, preset, codec string, crf int) (err error) {
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", inputPath,
		"-vf", "scale='" + scale + "',crop='iw-mod(iw,2)':'ih-mod(ih,2)'",
		"-vcodec", codec,
		"-crf", fmt.Sprint(crf),
		"-c:a", "copy",
		"-preset", preset,
		"-map_metadata", "0",
		"-movflags", "use_metadata_tags",
		outputPath,
	}

	execCmd := exec.CommandContext(ctx, f.binPath, args...) //nolint:gosec
	patchCmd(execCmd)

	f.logger.Debug(execCmd.String())

	output, err := f.cmd.Run(execCmd)
	if ctx.Err() != nil {
		return ctx.Err()
	} else if err != nil {
		return fmt.Errorf("%w: %s", ErrConversion, output)
	}
	return nil
}
