package ffmpeg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mjpegCodec is one of the encoders the output of FFMPEG listed by the fixtures
// holds, and one tinier never encodes videos with.
const mjpegCodec = "mjpeg"

// encodersOutput is an extract of the output of the -encoders option of FFMPEG,
// kept as close to the real thing as possible: the title, the lines explaining
// the flags, the separator and a handful of encoders of each kind.
const encodersOutput = `Encoders:
 V..... = Video
 A..... = Audio
 S..... = Subtitle
 .F.... = Frame-level multithreading
 ..S... = Slice-level multithreading
 ...X.. = Codec is experimental
 ....B. = Supports draw_horiz_band
 .....D = Supports direct rendering method 1
 ------
 V....D a64multi             Multicolor charset for Commodore 64 (codec a64_multi)
 VFS... mjpeg                MJPEG (Motion JPEG)
 V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)
 V....D libx264rgb           libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 RGB (codec h264)
 V....D libx265              libx265 H.265 / HEVC (codec hevc)
 V....D libvpx-vp9           libvpx VP9 (codec vp9)
 V..... png                  PNG image (codec png)
 A....D libopus              libopus Opus (codec opus)
 A..... aac                  AAC (Advanced Audio Coding)
 S..... text                 Raw text subtitle
`

func Test_parseEncoders(t *testing.T) {
	names := parseEncoders(encodersOutput)

	assert.Equal(t, []string{
		"a64multi", mjpegCodec, h264Codec, "libx264rgb", h265Codec,
		"libvpx-vp9", "png", "libopus", "aac", "text",
	}, names)
	assert.NotContains(t, names, "=", "the lines explaining the flags are not encoders")
}

func TestPickVideoCodec(t *testing.T) {
	tests := map[string]struct {
		wanted         string
		encoders       []string
		wantCodec      string
		wantFoundValue bool
	}{
		"the wanted codec is kept when it is available": {
			wanted:    svtAV1Codec,
			encoders:  []string{mjpegCodec, svtAV1Codec, h264Codec},
			wantCodec: svtAV1Codec, wantFoundValue: true,
		},
		"a missing codec falls back on the fastest encoder available": {
			wanted:    svtAV1Codec,
			encoders:  []string{mjpegCodec, h264Codec, h265Codec},
			wantCodec: h265Codec, wantFoundValue: true,
		},
		"libaom-av1 is too slow to be fallen back on": {
			wanted:         svtAV1Codec,
			encoders:       []string{mjpegCodec, "libaom-av1"},
			wantCodec:      svtAV1Codec,
			wantFoundValue: false,
		},
		"a codec given by the user is used when it is available": {
			wanted:    h264Codec,
			encoders:  []string{h264Codec, h265Codec},
			wantCodec: h264Codec, wantFoundValue: true,
		},
		"a codec given by the user falls back when it is not available": {
			wanted:    "nvenc",
			encoders:  []string{h265Codec, mjpegCodec},
			wantCodec: h265Codec, wantFoundValue: true,
		},
		"nothing is found without any encoder": {
			wanted:         svtAV1Codec,
			encoders:       nil,
			wantCodec:      svtAV1Codec,
			wantFoundValue: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			codec, found := pickVideoCodec(test.wanted, test.encoders)
			assert.Equal(t, test.wantCodec, codec)
			assert.Equal(t, test.wantFoundValue, found)
		})
	}
}
