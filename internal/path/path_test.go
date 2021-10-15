package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InputToOutput(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		inputPath     string
		outputDirPath string
		outExt        string
		outputPath    string
	}{
		"Windows path": {
			inputPath:     `input\\100andro\\mov_0017.mp4`,
			outputDirPath: `C:\output`,
			outputPath:    `C:\output/100andro/mov_0017.mp4`,
		},
		"Nix path": {
			inputPath:     `input/100andro/mov_0017.mp4`,
			outputDirPath: `/output`,
			outputPath:    `/output/100andro/mov_0017.mp4`,
		},
		"output at current path": {
			inputPath:     `input/100andro/mov_0017.mp4`,
			outputDirPath: ``,
			outputPath:    `100andro/mov_0017.mp4`,
		},
		"output at dot": {
			inputPath:     `input/100andro/mov_0017.mp4`,
			outputDirPath: `.`,
			outputPath:    `100andro/mov_0017.mp4`,
		},
		"output with output extension set": {
			inputPath:     `input/100andro/mov_0017.mp4`,
			outputDirPath: `output`,
			outExt:        ".mov",
			outputPath:    `output/100andro/mov_0017.mov`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outputPath := InputToOutput(testCase.inputPath,
				testCase.outputDirPath, testCase.outExt)

			assert.Equal(t, testCase.outputPath, outputPath)
		})
	}
}
