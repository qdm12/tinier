package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/qdm12/tinier/internal/spinner"
	"golang.org/x/sys/cpu"
)

func download(ctx context.Context, httpClient HTTPClient, stdout io.Writer,
	destinationDir string) (absolutePath string, err error) {
	const line = "📥 Downloading ffmpeg..."
	spinner := spinner.New(stdout, line, []rune{'⌛', '⏳', '📥'})
	spinner.Start()

	url, err := getURLForPlatform()
	if err != nil {
		return "", fmt.Errorf("getting ffmpeg URL: %w", err)
	}

	err = emptyDir(destinationDir)
	if err != nil {
		return "", fmt.Errorf("emptying ffmpeg cache directory: %w", err)
	}

	var extract extractFunc
	switch {
	case strings.HasSuffix(url, ".tar.xz"):
		extract = extractTarXZ
	case strings.HasSuffix(url, ".zip"):
		extract = extractZip
	default:
		panic(fmt.Sprintf("unsupported URL file extension: %s", url))
	}

	err = downloadAndExtract(ctx, httpClient, url, destinationDir, extract)
	if err != nil {
		return "", fmt.Errorf("downloading and extracting: %w", err)
	}
	spinner.Stop()
	fmt.Fprintln(stdout, "✔️")

	absolutePath, err = searchFileRecursive(destinationDir, "ffmpeg", "ffmpeg.exe")
	if err != nil {
		return "", fmt.Errorf("searching ffmpeg binary: %w", err)
	} else if absolutePath == "" {
		return "", fmt.Errorf("%w: in %s", ErrFFMPEGBinNotFound, destinationDir)
	}

	const binMode = os.FileMode(0755)
	err = os.Chmod(absolutePath, binMode)
	if err != nil {
		return "", fmt.Errorf("setting ffmpeg permissions: %w", err)
	}

	return absolutePath, nil
}

var ErrPlatformNotSupported = errors.New("platform not supported")

func getURLForPlatform() (httpURL string, err error) {
	switch runtime.GOOS {
	case "linux":
		const johnVanSickleFormatURL = "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-%s-static.tar.xz"
		switch runtime.GOARCH {
		case "amd64":
			return fmt.Sprintf(johnVanSickleFormatURL, "amd64"), nil
		case "arm64":
			return fmt.Sprintf(johnVanSickleFormatURL, "arm64"), nil
		case "386":
			return fmt.Sprintf(johnVanSickleFormatURL, "i686"), nil
		case "arm":
			switch {
			case cpu.ARM.HasVFPv3, cpu.ARM.HasVFP: // armv7 or armv6
				return fmt.Sprintf(johnVanSickleFormatURL, "armhf"), nil
			default: // armv5
				return fmt.Sprintf(johnVanSickleFormatURL, "armel"), nil
			}
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n5.1-latest-win64-gpl-5.1.zip", nil
		}
	}

	return "", fmt.Errorf("%w: %s %s",
		ErrPlatformNotSupported, runtime.GOOS, runtime.GOARCH)
}

type extractFunc func(reader io.Reader, directoryPath string) (err error)

func downloadAndExtract(ctx context.Context, client HTTPClient,
	url string, destinationDir string, extract extractFunc) (err error) {
	const dirPerms = os.FileMode(0744)
	err = os.MkdirAll(destinationDir, dirPerms)
	if err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	err = extract(response.Body, destinationDir)
	if err != nil {
		_ = response.Body.Close()
		return fmt.Errorf("extracting tar.gz: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	return nil
}
