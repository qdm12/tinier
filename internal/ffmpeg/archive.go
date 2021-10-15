package ffmpeg

import (
	"fmt"
	"io"
	"os"
)

func writeReaderToFile(reader io.Reader, destinationPath string, perms os.FileMode) (err error) {
	targetFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}

	_, err = io.Copy(targetFile, reader)
	if err != nil {
		return fmt.Errorf("extracting file data to %s: %w",
			destinationPath, err)
	}

	err = targetFile.Close()
	if err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return nil
}
