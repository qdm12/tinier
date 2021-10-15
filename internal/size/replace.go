package size

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrOpenFileToReplace    = errors.New("failed to open the file to replace")
	ErrCloseFileToReplace   = errors.New("failed to close the file to replace")
	ErrOpenReplacementFile  = errors.New("failed to open the replacement file")
	ErrCloseReplacementFile = errors.New("failed to close the replacement file")
	ErrCopyFromReplacement  = errors.New("failed to copy from replacement file")
)

func ReplaceBy(toReplacePath, replacementPath string) (err error) {
	const fileMode os.FileMode = 0600
	toReplaceFile, err := os.OpenFile(toReplacePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, fileMode)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrOpenFileToReplace, err)
	}

	replacementFile, err := os.Open(replacementPath)
	if err != nil {
		_ = toReplaceFile.Close()
		return fmt.Errorf("%w: %s", ErrOpenReplacementFile, err)
	}

	_, err = io.Copy(toReplaceFile, replacementFile)
	if err != nil {
		_ = toReplaceFile.Close()
		_ = replacementFile.Close()
		return fmt.Errorf("%w: %s", ErrCopyFromReplacement, err)
	}

	if err := toReplaceFile.Close(); err != nil {
		return fmt.Errorf("%w: %s", ErrCloseFileToReplace, err)
	}

	if err := replacementFile.Close(); err != nil {
		return fmt.Errorf("%w: %s", ErrCloseReplacementFile, err)
	}

	return nil
}
