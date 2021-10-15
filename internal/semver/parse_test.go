package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Extract(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		s          string
		expected   Semver
		errWrapped error
		errMessage string
	}{
		"semver with extra suffix": {
			s: "1.2.3-alpha.1",
			expected: Semver{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
		},
		"partial semver with extra suffix": {
			s: "1.2-alpha.1",
			expected: Semver{
				Major: 1,
				Minor: 2,
			},
		},
		"no semver": {
			s:          "alpha.1",
			errWrapped: ErrSemverNotFound,
			errMessage: "semver not found: in \"alpha.1\"",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			version, err := Extract(testCase.s)

			assert.Equal(t, testCase.expected, version)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}
