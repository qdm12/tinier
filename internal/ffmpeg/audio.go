package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
)

func (f *FFMPEG) TinyAudio(ctx context.Context, inputPath, outputPath,
	codec string, qScale uint, bitRate string) (err error) {
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", inputPath,
		"-acodec", codec,
	}

	if codec == "libopus" {
		args = append(args, "-compression_level", "10") // favor quality over compression speed
		args = append(args, "-frame_duration", "60")    // better quality for 40ms latency
	}

	// Either use bitrate or qscale
	if bitRate != "" {
		args = append(args, "-b:a", bitRate)
	} else {
		args = append(args, "-qscale:a", fmt.Sprint(qScale))
	}

	args = append(args,
		"-map_metadata", "0",
		"-movflags", "use_metadata_tags")

	args = append(args, outputPath)

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
