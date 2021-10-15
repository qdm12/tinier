package size

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BytesToHuman(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		bytes int64
		s     string
	}{
		"0": {
			s: "0B",
		},
		"100": {
			bytes: 100,
			s:     "100B",
		},
		"7KB": {
			bytes: 8000,
			s:     "7KB",
		},
		"7MB": {
			bytes: 8000000,
			s:     "7MB",
		},
		"7GB": {
			bytes: 8000000000,
			s:     "7GB",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := BytesToHuman(testCase.bytes)

			assert.Equal(t, testCase.s, s)
		})
	}
}
