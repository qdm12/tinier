package path

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// InputToOutput returns a temporary output path and a final output path for a given input path.
// `outputPath` = `outputDirPath`/<path to `inputPath`>/<input_filename>(.outExt)
// `outputTempPath` = `outputDirPath`/<path to `inputPath`>/tmp_<input_filename>(.outExt).
func InputToOutput(inputPath, outputDirPath, outputExt string) (outputTempPath, outputPath string) {
	inputPath = filepath.Clean(inputPath)
	inputPath = strings.ReplaceAll(inputPath, "\\", string(os.PathSeparator))

	if outputExt != "" {
		inputExt := filepath.Ext(inputPath)
		inputPath = strings.TrimSuffix(inputPath, inputExt)
		inputPath += outputExt
	}

	pathParts := strings.Split(inputPath, string(os.PathSeparator))
	pathWithoutRootDir := strings.Join(pathParts[1:], string(os.PathSeparator))
	outputPath = filepath.Join(outputDirPath, pathWithoutRootDir)

	tempOutputFilename := filepath.Base(outputPath)
	tempOutputFilename = "tmp_" + tempOutputFilename
	outputTempPath = filepath.Join(filepath.Dir(outputPath), tempOutputFilename)

	return outputTempPath, outputPath
}

func Walk(rootDir string, imageExtensions, audioExtensions, videoExtensions []string) (
	imagePaths, audioPaths, videoPaths, otherPaths []string, err error) {
	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		loweredPath := strings.ToLower(path) // for extension purpose, we lower case everything
		switch {
		case suffixIsOneOf(loweredPath, imageExtensions...):
			imagePaths = append(imagePaths, path)
		case suffixIsOneOf(loweredPath, audioExtensions...):
			audioPaths = append(audioPaths, path)
		case suffixIsOneOf(loweredPath, videoExtensions...):
			videoPaths = append(videoPaths, path)
		case d.IsDir(): // ignore directories
		default:
			otherPaths = append(otherPaths, path)
		}
		return nil
	})
	return imagePaths, audioPaths, videoPaths, otherPaths, err
}

func suffixIsOneOf(s string, suffixes ...string) (ok bool) {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}
