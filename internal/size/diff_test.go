package size

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DiffString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		outputSize int64
		inputSize  int64
		s          string
	}{
		"same output": {
			outputSize: 11000,
			inputSize:  11000,
			s:          "10KB ➡️  10KB (+0.0%)",
		},
		"bigger output": {
			outputSize: 11000,
			inputSize:  9000,
			s:          "8KB ➡️  10KB (+22.2%)",
		},
		"smaller output": {
			outputSize: 9000,
			inputSize:  11000,
			s:          "10KB ➡️  8KB (-18.2%)",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := DiffString(testCase.outputSize, testCase.inputSize)

			assert.Equal(t, testCase.s, s)
		})
	}
}
