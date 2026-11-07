package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageSetDefaults(t *testing.T) {
	var image Image
	image.setDefaults()

	// An image of 1280 pixels wide or less is left to its own width, instead of
	// being upscaled to 1280 pixels.
	assert.Equal(t, defaultImageScale, image.Scale)
	// The AVIF files of a phone hold a still photo and a short movie, and a JPEG
	// file can hold neither the transparency nor the high dynamic range of them:
	// AVIF files are left as they are, unless their extension is given.
	assert.Equal(t, []string{".jpg", ".jpeg", ".png"}, image.Extensions)
	assert.Equal(t, ".jpg", image.OutputExtension)
}

func TestImageValidateScale(t *testing.T) {
	tests := []struct {
		scale string
		valid bool
	}{
		{scale: defaultImageScale, valid: true},
		{scale: "min(1280,iw):720", valid: true},
		{scale: "max(720,ih):-1", valid: true},
		{scale: "1280:-1", valid: true},
		{scale: "min(1280):-1", valid: false},
		{scale: "min(1280,iw,ih):-1", valid: false},
		{scale: "1280:720 garbage", valid: false},
		{scale: "a:b", valid: false},
		{scale: "", valid: false},
	}

	for _, test := range tests {
		var image Image
		image.setDefaults()
		image.Scale = test.scale

		err := image.validate()

		if test.valid {
			assert.NoError(t, err, "scale %q", test.scale)
		} else {
			assert.Error(t, err, "scale %q", test.scale)
		}
	}
}
