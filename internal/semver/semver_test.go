package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Semver_Before(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		semver Semver
		other  Semver
		before bool
	}{
		"semver before other": {
			semver: Semver{
				Major: 5,
				Minor: 0,
				Patch: 0,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
			before: true,
		},
		"semver equal other": {
			semver: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
		},
		"semver after other": {
			semver: Semver{
				Major: 5,
				Minor: 1,
				Patch: 0,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			before := testCase.semver.Before(testCase.other)

			assert.Equal(t, testCase.before, before)
		})
	}
}

func Test_Semver_After(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		semver Semver
		other  Semver
		after  bool
	}{
		"semver before other": {
			semver: Semver{
				Major: 5,
				Minor: 0,
				Patch: 0,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
		},
		"semver equal other": {
			semver: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
		},
		"semver after other": {
			semver: Semver{
				Major: 5,
				Minor: 1,
				Patch: 0,
			},
			other: Semver{
				Major: 5,
				Minor: 0,
				Patch: 1,
			},
			after: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			after := testCase.semver.After(testCase.other)

			assert.Equal(t, testCase.after, after)
		})
	}
}
