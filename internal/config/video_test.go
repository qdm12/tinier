package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoSetDefaults(t *testing.T) {
	t.Run("the defaults are the ones of the default encoder", func(t *testing.T) {
		var video Video
		video.setDefaults()

		assert.Equal(t, "libsvtav1", video.Codec)
		assert.Equal(t, "8", video.Preset)
		// A video smaller than 1280 pixels wide is left to its own width,
		// instead of being upscaled to 1280 pixels.
		assert.Equal(t, "min(1280,iw):-2", video.Scale)
	})

	t.Run("the default preset follows the encoder given", func(t *testing.T) {
		tests := []struct {
			codec      string
			wantPreset string
		}{
			{codec: "libx264", wantPreset: "fast"},
			{codec: "libx265", wantPreset: "fast"},
			{codec: "libsvtav1", wantPreset: "8"},
			{codec: "libaom-av1", wantPreset: "8"},
		}

		for _, test := range tests {
			var video Video
			video.Codec = test.codec
			video.setDefaults()

			assert.Equal(t, test.wantPreset, video.Preset,
				"preset of the encoder %s", test.codec)
		}
	})

	t.Run("a given preset is kept", func(t *testing.T) {
		var video Video
		video.Codec = "libx264"
		video.Preset = "slow"
		video.setDefaults()

		assert.Equal(t, "slow", video.Preset)
	})
}

func TestVideoValidateScale(t *testing.T) {
	tests := []struct {
		scale string
		valid bool
	}{
		{scale: "min(1280,iw):-2", valid: true},
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

func TestImageValidateScale(t *testing.T) {
	var image Image
	image.setDefaults()
	image.Scale = "min(1280,iw):-2"

	err := image.validate()

	require.NoError(t, err)
}
