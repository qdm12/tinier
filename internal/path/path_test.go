package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testInputPath is the path of a video file, in a directory tree, used by the
// test cases of this package.
const testInputPath = `input/100andro/mov_0017.mp4`

func Test_InputToOutput(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		inputPath      string
		outputDirPath  string
		outExt         string
		outputTempPath string
		outputPath     string
	}{
		"Windows path": {
			inputPath:      `input\\100andro\\mov_0017.mp4`,
			outputDirPath:  `C:\output`,
			outputTempPath: `C:\output/100andro/tmp_mov_0017.mp4`,
			outputPath:     `C:\output/100andro/mov_0017.mp4`,
		},
		"Nix path": {
			inputPath:      testInputPath,
			outputDirPath:  `/output`,
			outputTempPath: `/output/100andro/tmp_mov_0017.mp4`,
			outputPath:     `/output/100andro/mov_0017.mp4`,
		},
		"output at current path": {
			inputPath:      testInputPath,
			outputDirPath:  ``,
			outputTempPath: `100andro/tmp_mov_0017.mp4`,
			outputPath:     `100andro/mov_0017.mp4`,
		},
		"output at dot": {
			inputPath:      testInputPath,
			outputDirPath:  `.`,
			outputTempPath: `100andro/tmp_mov_0017.mp4`,
			outputPath:     `100andro/mov_0017.mp4`,
		},
		"output with output extension set": {
			inputPath:      testInputPath,
			outputDirPath:  `output`,
			outExt:         ".mov",
			outputTempPath: `output/100andro/tmp_mov_0017.mov`,
			outputPath:     `output/100andro/mov_0017.mov`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outputTempPath, outputPath := InputToOutput(testCase.inputPath,
				testCase.outputDirPath, testCase.outExt)

			assert.Equal(t, testCase.outputTempPath, outputTempPath)
			assert.Equal(t, testCase.outputPath, outputPath)
		})
	}
}
