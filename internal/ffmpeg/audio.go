package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
)

func (f *FFMPEG) TinyAudio(ctx context.Context, inputPath, outputPath,
	codec string, qScale int) (err error) {
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", inputPath,
		"-acodec", codec,
		"-qscale:a", fmt.Sprint(qScale),
		"-map_metadata", "0",
		"-movflags", "use_metadata_tags",
		outputPath,
	}

	execCmd := exec.CommandContext(ctx, f.binPath, args...) //nolint:gosec
	patchCmd(execCmd)

	output, err := f.cmd.Run(execCmd)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrConversion, output)
	}
	return nil
}
