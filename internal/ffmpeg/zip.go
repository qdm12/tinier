package ffmpeg

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func extractZip(zipStream io.Reader, destinationPath string) (err error) {
	zipBytes, err := io.ReadAll(zipStream)
	if err != nil {
		return fmt.Errorf("reading zip stream: %w", err)
	}

	buffer := bytes.NewReader(zipBytes)
	reader, err := zip.NewReader(buffer, int64(buffer.Len()))
	if err != nil {
		return fmt.Errorf("creating zip reader: %w", err)
	}

	for _, file := range reader.File {
		err = extractZipFile(file, destinationPath)
		if err != nil {
			return fmt.Errorf("extracting zip file: %w", err)
		}
	}

	return nil
}

func extractZipFile(file *zip.File, destinationPath string) (err error) {
	filePath, err := sanitizeArchivePath(destinationPath, file.Name)
	if err != nil {
		return fmt.Errorf("sanitizing archive path: %w", err)
	}

	permissions := file.Mode()

	if file.FileInfo().IsDir() {
		err = os.MkdirAll(filePath, permissions)
		if err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
		return nil
	}

	const directoryPermissions = os.FileMode(0744)
	err = os.MkdirAll(filepath.Dir(filePath), directoryPermissions)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	archiveFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("opening archive file: %w", err)
	}

	err = writeReaderToFile(archiveFile, filePath, permissions)
	if err != nil {
		return fmt.Errorf("extracting file: %w", err)
	}

	err = archiveFile.Close()
	if err != nil {
		return fmt.Errorf("closing archive file: %w", err)
	}

	return nil
}
