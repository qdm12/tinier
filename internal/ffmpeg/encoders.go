package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// fallbackVideoCodecs are the video encoders to fall back on, in order of
// preference, when the wanted one is missing from the FFMPEG build. The
// libaom-av1 encoder is left out of the list: it is an order of magnitude
// slower than the others, which makes it a poor replacement for a missing
// encoder.
//
//nolint:gochecknoglobals // ordered list of fixed encoders, never mutated
var fallbackVideoCodecs = []string{svtAV1Codec, h265Codec, h264Codec}

// Names of the video encoders tinier encodes with: the one it defaults to, and
// the ones it falls back on when the build of FFMPEG does not provide it.
const (
	svtAV1Codec = "libsvtav1"
	h264Codec   = "libx264"
	h265Codec   = "libx265"
)

// encoderRegexp matches an encoder of the output of the -encoders option of
// FFMPEG, which looks like " V....D libx264  libx264 H.264 (codec h264)": the
// type of media, 5 flags, then the name of the encoder. The lines explaining
// the flags, like " V..... = Video", are not matched because the name of an
// encoder never starts with an equal sign.
var encoderRegexp = regexp.MustCompile(`^ [VAPS]\S{5} ([^=\s]\S*)`)

// Encoders returns the names of the encoders the FFMPEG binary provides, such
// as libx264 or mjpeg.
func (f *FFMPEG) Encoders(ctx context.Context) (names []string, err error) {
	//nolint:gosec // the binary path is the one of the FFMPEG build tinier selected
	cmd := exec.CommandContext(ctx, f.binPath, "-hide_banner", "-encoders")
	patchCmd(cmd)
	output, err := f.cmd.Run(cmd)
	if err != nil {
		return nil, err
	}
	return parseEncoders(output), nil
}

// PickVideoCodec returns the video codec to encode with: the wanted one when
// this FFMPEG binary provides it, or the first encoder tinier falls back on.
// The wanted codec is kept, and false is returned, when the binary provides
// neither it nor any fallback, in which case a video conversion would fail.
func (f *FFMPEG) PickVideoCodec(ctx context.Context,
	wanted string) (codec string, found bool, err error) {
	encoders, err := f.Encoders(ctx)
	if err != nil {
		return wanted, false, fmt.Errorf("listing the encoders of FFMPEG: %w", err)
	}
	codec, found = pickVideoCodec(wanted, encoders)
	return codec, found, nil
}

// pickVideoCodec returns the video codec to encode with between the wanted one
// and the fallbacks of tinier, the first encoder the FFMPEG build provides.
// The wanted codec is kept, and false is returned, when the FFMPEG build
// provides none of them.
func pickVideoCodec(wanted string, encoders []string) (codec string, found bool) {
	if slices.Contains(encoders, wanted) {
		return wanted, true
	}
	for _, fallback := range fallbackVideoCodecs {
		if slices.Contains(encoders, fallback) {
			return fallback, true
		}
	}
	return wanted, false
}

// parseEncoders returns the names of the encoders of the output of the
// -encoders option of FFMPEG.
func parseEncoders(output string) (names []string) {
	for _, line := range strings.Split(output, "\n") {
		match := encoderRegexp.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		names = append(names, match[1])
	}
	return names
}
