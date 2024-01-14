package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var (
	ErrCodecUnsupported = errors.New("codec unsupported")
	ErrConversion       = errors.New("failed FFMPEG conversion")
)

func (f *FFMPEG) TinyImage(ctx context.Context, inputPath, outputPath,
	codec, scale string, crf, qScale uint) (err error) {
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", inputPath,
		"-vf", "scale=" + scale,
		"-map_metadata", "0",
		"-movflags", "use_metadata_tags",
		"-c:v", codec,
	}

	switch codec {
	case "libaom-av1":
		args = append(args,
			"-still-picture", "1",
			"-crf", fmt.Sprint(crf))
	case "mjpeg":
		args = append(args,
			"-qscale:v", fmt.Sprint(qScale))
	default:
		return fmt.Errorf("%w: %s", ErrCodecUnsupported, codec)
	}

	args = append(args, outputPath, "-noautorotate")

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
