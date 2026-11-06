package main

import (
	"testing"

	"github.com/qdm12/gosettings/reader/sources/env"
	"github.com/stretchr/testify/assert"
)

// Environment variable of the fixtures of this file, holding the constant rate
// factor of the images, and the key of the settings it sets.
const (
	imageCRFVariable = "TINIER_IMAGE_CRF=40"
	imageCRFKey      = "IMAGE_CRF"
)

func Test_withoutKeyPrefix(t *testing.T) {
	t.Parallel()

	environ := []string{
		imageCRFVariable,
		"TINIER_FFMPEG_PATH=",
		"PATH=/bin:/usr/bin",
		"NO_EQUAL_SIGN",
	}

	stripped := withoutKeyPrefix(environ, keyPrefix)

	assert.Equal(t, []string{
		"IMAGE_CRF=40",
		"FFMPEG_PATH=",
		"PATH=/bin:/usr/bin",
	}, stripped)
}

// Test_environmentSourceDoublesTheKeyPrefix documents why the key prefix is
// removed from the environment of the process before being handed over to the
// environment source of gosettings: this source adds the prefix to every name it
// reads, and to every key it is asked for, so a prefixed variable of the process
// is looked up under a doubled prefix, where it cannot be found.
func Test_environmentSourceDoublesTheKeyPrefix(t *testing.T) {
	t.Parallel()

	source := env.New(env.Settings{
		Environ:   []string{imageCRFVariable},
		KeyPrefix: keyPrefix,
	})

	value, isSet := readKey(source, imageCRFKey)

	assert.False(t, isSet, "the value of TINIER_IMAGE_CRF is stored under TINIER_TINIER_IMAGE_CRF")
	assert.Empty(t, value)
}

// Test_readPrefixedVariable verifies the environment variables of the process,
// prefixed or not, are the ones setting a key of the settings.
func Test_readPrefixedVariable(t *testing.T) {
	tests := map[string]struct {
		environ []string
		key     string
		value   string
	}{
		"prefixed": {
			environ: []string{imageCRFVariable},
			key:     imageCRFKey,
			value:   "40",
		},
		"prefixed with a dash": {
			environ: []string{"TINIER_FFMPEG_PATH=/bin/ffmpeg"},
			key:     "FFMPEG_PATH",
			value:   "/bin/ffmpeg",
		},
		"not prefixed": {
			environ: []string{"IMAGE_CRF=41"},
			key:     "IMAGE_CRF",
			value:   "41",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			source := env.New(env.Settings{
				Environ:   withoutKeyPrefix(test.environ, keyPrefix),
				KeyPrefix: keyPrefix,
			})

			value, isSet := readKey(source, test.key)

			assert.True(t, isSet)
			assert.Equal(t, test.value, value)
		})
	}
}

// readKey returns the value the environment source holds for the given key,
// which is how the reader of the settings asks a source for a value.
func readKey(source *env.Source, key string) (value string, isSet bool) {
	return source.Get(source.KeyTransform(key))
}
