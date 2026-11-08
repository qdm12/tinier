package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowPreset is a preset of the x264 and x265 encoders, used by the fixtures.
const slowPreset = "slow"

func TestVideoSetDefaults(t *testing.T) {
	t.Run("the defaults leave the preset to the encoder", func(t *testing.T) {
		var video Video
		video.setDefaults()

		assert.Equal(t, defaultVideoCodec, video.Codec)
		// A video smaller than 1280 pixels wide is left to its own width,
		// instead of being upscaled to 1280 pixels.
		assert.Equal(t, defaultVideoScale, video.Scale)
		// A preset is encoder specific, and which encoder converts is only known
		// once the build of FFMPEG has been read: the default asks the encoder for
		// its own, and is not the zero value of the field.
		assert.Equal(t, PresetAuto, video.Preset)
	})

	t.Run("a given preset is kept", func(t *testing.T) {
		var video Video
		video.Codec = h264Codec
		video.Preset = slowPreset
		video.setDefaults()

		assert.Equal(t, slowPreset, video.Preset)
	})
}

func TestVideoValidateScale(t *testing.T) {
	tests := []struct {
		scale string
		valid bool
	}{
		{scale: defaultVideoScale, valid: true},
		{scale: "min(1280,iw):720", valid: true},
		{scale: "max(720,ih):-1", valid: true},
		{scale: "1280:-1", valid: true},
		{scale: "1280:720", valid: true},
		{scale: "640:-2", valid: true},
		{scale: "min(1280):-1", valid: false},
		{scale: "min(1280,iw,ih):-1", valid: false},
		{scale: "1280:720 garbage", valid: false},
		{scale: "a:b", valid: false},
		{scale: "", valid: false},
	}

	for _, test := range tests {
		var video Video
		video.setDefaults()
		video.Scale = test.scale

		err := video.validate()

		if test.valid {
			assert.NoError(t, err, "scale %q", test.scale)
		} else {
			assert.Error(t, err, "scale %q", test.scale)
		}
	}
}

func TestValidatePreset(t *testing.T) {
	tests := map[string]struct {
		codec  string
		preset string
		valid  bool
	}{
		"a number is a preset of SVT-AV1": {
			codec: defaultVideoCodec, preset: "8", valid: true,
		},
		"a word is not a preset of SVT-AV1": {
			codec: defaultVideoCodec, preset: fastPreset, valid: false,
		},
		"a word is a preset of x264": {
			codec: h264Codec, preset: fastPreset, valid: true,
		},
		"a word is a preset of x265": {
			codec: h265Codec, preset: slowPreset, valid: true,
		},
		"a number is not a preset of x264": {
			codec: h264Codec, preset: "8", valid: false,
		},
		"a number is not a preset of the AV1 encoder of libaom": {
			codec: aomAV1Codec, preset: "8", valid: false,
		},
		"any preset is kept for an encoder tinier does not know": {
			codec: "libkvazaar", preset: "4", valid: true,
		},
		"auto is the preset of any encoder": {
			codec: h264Codec, preset: PresetAuto, valid: true,
		},
		"auto is the preset of SVT-AV1 too": {
			codec: defaultVideoCodec, preset: PresetAuto, valid: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validatePreset(test.codec, test.preset)

			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}

	t.Run("the settings of a codec and of the auto preset go together", func(t *testing.T) {
		var video Video
		video.setDefaults()
		video.Codec = h264Codec

		require.NoError(t, video.validate())
	})

	t.Run("the fastest preset of SVT-AV1 is accepted", func(t *testing.T) {
		var video Video
		video.setDefaults()

		require.NoError(t, validatePreset(video.Codec, "13"))
	})
}

// The preset tinier resolves for the auto preset has to be one of the presets
// of the encoder which converts, or FFMPEG would refuse to open it: a preset
// unknown to the encoder is refused by the settings, whether it comes from a
// user or from tinier itself.
func TestVideoDefaultPreset(t *testing.T) {
	tests := map[string]struct {
		codec      string
		wantPreset string
	}{
		"the encoder tinier defaults to takes a number": {
			codec: defaultVideoCodec, wantPreset: svtAV1Preset,
		},
		"the x264 encoder takes a word": {
			codec: h264Codec, wantPreset: fastPreset,
		},
		"the x265 encoder takes a word": {
			codec: h265Codec, wantPreset: fastPreset,
		},
		"the AV1 encoder of libaom takes a word": {
			codec: aomAV1Codec, wantPreset: fastPreset,
		},
		"an encoder tinier does not know takes a number": {
			codec: "libkvazaar", wantPreset: svtAV1Preset,
		},
		"the case of the name of the encoder is of no consequence": {
			codec: "LibX265", wantPreset: fastPreset,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			video := Video{Codec: test.codec}

			assert.Equal(t, test.wantPreset, video.DefaultPreset())
		})
	}

	t.Run("the preset of an encoder is one of its presets", func(t *testing.T) {
		codecs := []string{defaultVideoCodec, h264Codec, h265Codec, aomAV1Codec}

		for _, codec := range codecs {
			video := Video{Codec: codec}

			preset := video.DefaultPreset()
			err := validatePreset(codec, preset)

			assert.NoError(t, err, "preset %q of the encoder %s", preset, codec)
		}
	})
}

// A preset valid for one encoder is refused for the next one: the settings only
// take a preset the encoder of the conversion can open with.
func TestVideoPresetAndEncoderGoTogether(t *testing.T) {
	codecsAndPresets := []struct {
		codec  string
		preset string
	}{
		{defaultVideoCodec, svtAV1Preset},
		{h264Codec, fastPreset},
		{h265Codec, slowPreset},
		{aomAV1Codec, fastPreset},
	}

	for _, codecAndPreset := range codecsAndPresets {
		var video Video
		video.setDefaults()
		video.Codec = codecAndPreset.codec
		video.Preset = codecAndPreset.preset

		err := video.validate()

		assert.NoError(t, err, "preset %q of the encoder %s",
			codecAndPreset.preset, codecAndPreset.codec)
	}
}
