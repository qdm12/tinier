package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/qdm12/tinier/internal/semver"
)

var ErrFFMPEGBinNotFound = errors.New("ffmpeg binary file not found")

// SetupFFMPEG sets up ffmpeg if needed and returns an absolute
// path to the `ffmpeg` binary to use.
// It first checks for the user provided ffmpeg path and verify
// its version is at least the minimum version given.
// If not, it searches for a PATH available `ffmpeg` and verify
// its version is at least the minimum version given.
// Finally, if both check fail, it falls back to downloading the
// latest ffmpeg binary for the current platform.
func SetupFFMPEG(ctx context.Context, minVersion semver.Semver, //nolint:cyclop
	userFfmpegPath string, runner Runner, httpClient HTTPClient,
	stdout io.Writer) (absolutePath string, err error) {
	fmt.Fprintf(stdout, "🔦 Looking for FFMPEG...")

	// Try user given ffmpeg path
	_, err = os.Stat(userFfmpegPath)
	if err == nil {
		err = checkFFMPEGValidity(ctx, userFfmpegPath, runner, minVersion)
		if err == nil {
			absolutePath, err = filepath.Abs(userFfmpegPath)
			if err != nil {
				return "", fmt.Errorf("getting absolute path: %w", err)
			}
			fmt.Fprintln(stdout, "✔️")
			return absolutePath, err
		}
		fmt.Fprintf(stdout, "\n⚠️ %s\n", err)
	}

	// Try ffmpeg binary from cache directory from a previous run
	// of tinier.
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache directory: %w", err)
	}

	tinierCacheDir := filepath.Join(cacheDir, "tinier")
	ffmpegCacheDir := filepath.Join(tinierCacheDir, "ffmpeg")
	absolutePath, err = searchFileRecursive(ffmpegCacheDir, "ffmpeg", "ffmpeg.exe")
	if err != nil {
		return "", fmt.Errorf("searching ffmpeg binary: %w", err)
	} else if absolutePath != "" {
		err = checkFFMPEGValidity(ctx, absolutePath, runner, minVersion)
		if err == nil {
			fmt.Fprintln(stdout, "✔️")
			return absolutePath, nil
		}
		fmt.Fprintf(stdout, "⚠️  %s\n", err)
	}

	// Try ffmpeg binary from PATH
	absolutePath, err = exec.LookPath("ffmpeg")
	if err == nil {
		err = checkFFMPEGValidity(ctx, absolutePath, runner, minVersion)
		if err == nil {
			fmt.Fprintln(stdout, "✔️")
			return absolutePath, nil
		}
		fmt.Fprintf(stdout, "⚠️  %s\n", err)
	}

	// Last resort: download ffmpeg to tinier cache directory
	absolutePath, err = download(ctx, httpClient, stdout, ffmpegCacheDir)
	if err != nil {
		return "", fmt.Errorf("downloading ffmpeg: %w", err)
	}

	version, err := getVersion(ctx, absolutePath, runner)
	if err != nil {
		return "", fmt.Errorf("getting version of %s: %w", absolutePath, err)
	}

	fmt.Fprintf(stdout, "✨ Using ffmpeg version %s at %s\n", version, absolutePath)

	return absolutePath, nil
}

var ErrFFMPEGVersionTooLow = fmt.Errorf("ffmpeg version does not meet minimum version requirement")

func checkFFMPEGValidity(ctx context.Context, binPath string,
	runner Runner, minSemver semver.Semver) (err error) {
	version, err := getVersion(ctx, binPath, runner)
	if err != nil {
		return fmt.Errorf("getting version of %s: %w", binPath, err)
	}

	if !version.Before(minSemver) {
		return nil
	}

	return fmt.Errorf("%w: for %s: version %s is below minimum version %s",
		ErrFFMPEGVersionTooLow, binPath, version, minSemver)
}

func emptyDir(dirPath string) (err error) {
	err = os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("removing directory: %w", err)
	}

	const perms = os.FileMode(0744)
	err = os.MkdirAll(dirPath, perms)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	return nil
}

func searchFileRecursive(dirPath string, fileNames ...string) (
	path string, err error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("reading directory: %w", err)
	}

	var dirPaths []string
	for _, entry := range entries {
		// Check files first then directories
		if entry.IsDir() {
			dirPath := filepath.Join(dirPath, entry.Name())
			dirPaths = append(dirPaths, dirPath)
			continue
		}

		for _, fileName := range fileNames {
			if entry.Name() == fileName {
				return filepath.Join(dirPath, entry.Name()), nil
			}
		}
	}

	for _, dirPath := range dirPaths {
		path, err = searchFileRecursive(dirPath, fileNames...)
		if err != nil {
			return "", err
		} else if path != "" {
			return path, nil
		}
	}

	return "", nil
}
