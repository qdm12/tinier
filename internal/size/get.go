package size

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrGetSizeOutput = errors.New("failed to get the size of the output")
	ErrGetSizeInput  = errors.New("failed to get the size of the input")
)

func GetSizes(inputPath, outpathPath string) (
	inputSize, outputSize int64, err error) {
	inputSize, err = getSize(inputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", ErrGetSizeInput, err)
	}

	outputSize, err = getSize(outpathPath)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", ErrGetSizeOutput, err)
	}

	return inputSize, outputSize, nil
}

func getSize(path string) (size int64, err error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}
