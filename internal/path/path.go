package path

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func InputToOutput(inputPath, outputDirPath, outputExt string) (outputPath string) {
	inputPath = filepath.Clean(inputPath)
	inputPath = strings.ReplaceAll(inputPath, "\\", "/")

	if outputExt != "" {
		inputExt := filepath.Ext(inputPath)
		inputPath = strings.TrimSuffix(inputPath, inputExt)
		inputPath += outputExt
	}

	pathParts := strings.Split(inputPath, "/")
	pathWithoutRootDir := strings.Join(pathParts[1:], "/")
	return filepath.Join(outputDirPath, pathWithoutRootDir)
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
