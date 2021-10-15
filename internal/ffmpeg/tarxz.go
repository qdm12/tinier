package ffmpeg

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

var (
	ErrTarHeaderNotSupported = errors.New("tar header not supported")
)

func extractTarXZ(tarXZStream io.Reader, destinationPath string) (err error) {
	uncompressedStream, err := xz.NewReader(tarXZStream)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		done, err := extractNextTarXZ(tarReader, destinationPath)
		if err != nil {
			return err
		} else if done {
			break
		}
	}

	return nil
}

func extractNextTarXZ(tarReader *tar.Reader, destinationPath string) (done bool, err error) {
	header, err := tarReader.Next()
	if errors.Is(err, io.EOF) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("reading next tar header: %w", err)
	}

	destinationPath, err = sanitizeArchivePath(destinationPath, header.Name)
	if err != nil {
		return false, fmt.Errorf("sanitizing archive path: %w", err)
	}

	permissions := header.FileInfo().Mode().Perm()

	switch header.Typeflag {
	case tar.TypeDir:
		err = os.Mkdir(destinationPath, permissions)
		if err != nil {
			return false, fmt.Errorf("creating directory: %w", err)
		}
		return false, nil
	case tar.TypeReg:
		err = writeReaderToFile(tarReader, destinationPath, permissions)
		return false, err
	default:
		return false, fmt.Errorf("%w: %v for destination path %s",
			ErrTarHeaderNotSupported, header.Typeflag, destinationPath)
	}
}

var ErrArchiveContentPathIsTainted = errors.New("archive content filepath is tainted")

func sanitizeArchivePath(destinationPath, archivePath string) (fullPath string, err error) {
	fullPath = filepath.Join(destinationPath, archivePath)
	if strings.HasPrefix(fullPath, filepath.Clean(destinationPath)) {
		return fullPath, nil
	}

	return "", fmt.Errorf("%w: %s", ErrArchiveContentPathIsTainted, archivePath)
}
