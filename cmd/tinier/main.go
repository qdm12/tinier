package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/reader/sources/env"
	"github.com/qdm12/gosettings/reader/sources/flag"
	"github.com/qdm12/log"
	"github.com/qdm12/tinier/internal/cmd"
	"github.com/qdm12/tinier/internal/config"
	"github.com/qdm12/tinier/internal/ffmpeg"
	"github.com/qdm12/tinier/internal/filetime"
	"github.com/qdm12/tinier/internal/metadata"
	"github.com/qdm12/tinier/internal/models"
	"github.com/qdm12/tinier/internal/path"
	"github.com/qdm12/tinier/internal/semver"
	"github.com/qdm12/tinier/internal/size"
	"github.com/qdm12/tinier/internal/spinner"
	"github.com/qdm12/tinier/internal/stats"
)

//nolint:gochecknoglobals
var (
	version = "unknown"
	commit  = "unknown"
	date    = "an unknown date"
)

// keyPrefix is the prefix of the environment variables setting the options of
// tinier, as documented in the README.
const keyPrefix = "TINIER_"

// withoutKeyPrefix returns the environment variables given, stripped of the
// given key prefix. The environment source of gosettings adds this prefix to
// every name it reads, and to every key it is asked for, so handing it
// os.Environ() as is would look the variables up under a doubled prefix, where
// none of them can ever be found.
func withoutKeyPrefix(environ []string, prefix string) (stripped []string) {
	for _, keyValue := range environ {
		name, value, found := strings.Cut(keyValue, "=")
		if !found {
			continue
		}
		stripped = append(stripped, strings.TrimPrefix(name, prefix)+"="+value)
	}
	return stripped
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	buildInfo := models.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	flag := flag.New(os.Args)
	env := env.New(env.Settings{
		Environ:   withoutKeyPrefix(os.Environ(), keyPrefix),
		KeyPrefix: keyPrefix,
	})
	reader := reader.New(reader.Settings{
		Sources: []reader.Source{flag, env},
	})

	errorCh := make(chan error)
	go func() {
		errorCh <- _main(ctx, buildInfo, reader, os.Stdout, os.Stdin,
			http.DefaultClient)
	}()

	select {
	case err := <-errorCh:
		close(errorCh)
		if err == nil { // expected exit
			os.Exit(0)
		}
		fmt.Println("Fatal error:", err)
		os.Exit(1)
	case <-ctx.Done():
		stop()
	}

	const shutdownGracePeriod = time.Second
	timer := time.NewTimer(shutdownGracePeriod)
	select {
	case <-errorCh:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		fmt.Println("Shutdown timed out")
	}

	os.Exit(1)
}

type HTTPClient interface {
	Do(request *http.Request) (response *http.Response, err error)
}

const (
	fileAlreadyExists = "✔️  file already exists"
)

//nolint:wrapcheck
func _main(ctx context.Context, buildInfo models.BuildInfo,
	reader *reader.Reader, stdout io.Writer, _ io.Reader,
	httpClient HTTPClient) error {
	versionMessage := fmt.Sprintf("🤖 Version %s (commit %s built on %s)",
		buildInfo.Version, buildInfo.Commit, buildInfo.Date)
	fmt.Fprintln(stdout, versionMessage)

	var settings config.Settings
	err := settings.Read(reader)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}
	settings.SetDefaults()
	err = settings.Validate()
	if err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	fmt.Fprintln(stdout, settings.String())

	// Log level parse error checked in settings validation.
	logLevel, _ := log.ParseLevel(settings.Log.Level)
	logger := log.New(log.SetLevel(logLevel))

	cmd := cmd.New()

	minVersion := semver.MustParse(settings.FfmpegMinVersion)

	ffmpegPath, err := ffmpeg.SetupFFMPEG(ctx, minVersion,
		*settings.FfmpegPath, cmd, httpClient, stdout)
	if err != nil {
		fmt.Fprintln(stdout, "❌")
		return fmt.Errorf("failed to setup ffmpeg: %w", err)
	}

	ffmpeg := ffmpeg.New(cmd, ffmpegPath, minVersion, logger)

	fmt.Fprintf(stdout, "📁 Reading input directory %s... ", settings.InputDirPath)
	imagePaths, audioPaths, videoPaths, otherPaths, err := path.Walk(
		settings.InputDirPath, settings.Image.Extensions,
		settings.Audio.Extensions, settings.Video.Extensions)
	if err != nil {
		fmt.Fprintln(stdout, "❌")
		return err
	}

	fmt.Fprintf(stdout,
		"%d image(s), %d audio file(s) and %d video(s) found\n",
		len(imagePaths), len(audioPaths), len(videoPaths))

	err = setVideoEncoder(ctx, &settings, ffmpeg, len(videoPaths), stdout)
	if err != nil {
		fmt.Fprintln(stdout, "❌")
		return err
	}

	fmt.Fprintf(stdout, "📁 Creating output directory %s if needed... ", settings.OutputDirPath)
	const dirPerms fs.FileMode = 0700
	err = os.MkdirAll(settings.OutputDirPath, dirPerms)
	if err != nil {
		fmt.Fprintln(stdout, "❌")
		return err
	}
	fmt.Fprintln(stdout, "✔️")

	stats := stats.New()
	defer stats.Finish(stdout)

	doOthers(ctx, settings, otherPaths, stats, stdout)
	if err = ctx.Err(); err != nil {
		return err
	}

	doAudios(ctx, settings, audioPaths, ffmpeg, stats, stdout)
	if err = ctx.Err(); err != nil {
		return err
	}

	doImages(ctx, settings, imagePaths, ffmpeg, stats, stdout)
	if err = ctx.Err(); err != nil {
		return err
	}

	doVideos(ctx, settings, videoPaths, ffmpeg, stats, stdout)
	return ctx.Err()
}

func doOthers(ctx context.Context, settings config.Settings,
	inputPaths []string, stats *stats.Stats, w io.Writer) {
	for _, inputPath := range inputPaths {
		fmt.Fprintf(w, "🗄️  Copying %s ... ", inputPath)

		outcome, err := doOther(settings, inputPath)
		if err != nil {
			stats.Failures++
			outcome += warnSignErr(err)
		}
		fmt.Fprintln(w, outcome)
		if ctx.Err() != nil { // program stopped by user
			return
		}
	}
}

func doImages(ctx context.Context, settings config.Settings,
	inputPaths []string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats,
	w io.Writer) {
	if *settings.Image.Skip {
		fmt.Fprintln(w, "⚠️ Skipping image files")
		return
	}
	for _, inputPath := range inputPaths {
		fmt.Fprintf(w, "🗜️  Tinying %s ... ", inputPath)
		outcome, err := doImage(ctx, settings, inputPath, ffmpeg, stats)
		if err != nil {
			stats.Failures++
			outcome += warnSignErr(err)
		}
		fmt.Fprintln(w, outcome)
		if ctx.Err() != nil { // program stopped by user
			return
		}
	}
}

func doAudios(ctx context.Context, settings config.Settings,
	inputPaths []string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats,
	w io.Writer) {
	if *settings.Audio.Skip {
		fmt.Fprintln(w, "⚠️ Skipping audio files")
		return
	}
	for _, inputPath := range inputPaths {
		fmt.Fprintf(w, "🗜️  Tinying %s ... ", inputPath)
		outcome, err := doAudio(ctx, settings, inputPath, ffmpeg, stats)
		if err != nil {
			stats.Failures++
			outcome += warnSignErr(err)
		}
		fmt.Fprintln(w, outcome)
		if ctx.Err() != nil { // program stopped by user
			return
		}
	}
}

func doVideos(ctx context.Context, settings config.Settings,
	inputPaths []string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats,
	w io.Writer) {
	if *settings.Video.Skip {
		fmt.Fprintln(w, "⚠️ Skipping video files")
		return
	}
	for _, inputPath := range inputPaths {
		outcome, err := doVideo(ctx, settings, inputPath, ffmpeg, stats, w)
		if err != nil {
			stats.Failures++
			outcome += "  ⚠️  " + err.Error()
		}
		fmt.Fprintln(w, outcome)
		if ctx.Err() != nil { // program stopped by user
			return
		}
	}
}

func doOther(settings config.Settings, inputPath string) (
	outcome string, err error) {
	_, outputPath := path.InputToOutput(inputPath, settings.OutputDirPath, "")

	if !*settings.OverrideOutput {
		exist, err := path.DoesFileExist(outputPath)
		if err != nil {
			return "", err
		} else if exist {
			return fileAlreadyExists, nil
		}
	}

	outputDir := filepath.Dir(outputPath)
	const dirPerms os.FileMode = 0700
	err = os.MkdirAll(outputDir, dirPerms)
	if err != nil {
		return "", fmt.Errorf("cannot create parent output directory: %w", err)
	}

	srcFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("cannot open input file: %w", err)
	}

	const filePerm os.FileMode = 0600
	dstFile, err := os.OpenFile(outputPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, filePerm)
	if err != nil {
		_ = srcFile.Close()
		return "", fmt.Errorf("cannot open output file: %w", err)
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		_ = srcFile.Close()
		_ = dstFile.Close()
		return "", fmt.Errorf("cannot copy: %w", err)
	}

	err = filetime.Copy(outputPath, inputPath)
	if err != nil {
		_ = os.Remove(outputPath) // clean up
		return "", err
	}

	return "✔️", nil
}

func doImage(ctx context.Context, settings config.Settings,
	inputPath string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats) (
	outcome string, err error) {
	_, outputPath := path.InputToOutput(inputPath,
		settings.OutputDirPath, settings.Image.OutputExtension)

	if !*settings.OverrideOutput {
		exist, err := path.DoesFileExist(outputPath)
		if err != nil {
			return "", err
		} else if exist {
			return fileAlreadyExists, nil
		}
	}

	outputDir := filepath.Dir(outputPath)
	const dirPerms os.FileMode = 0700
	err = os.MkdirAll(outputDir, dirPerms)
	if err != nil {
		return "", fmt.Errorf("cannot create parent output directory: %w", err)
	}

	err = ffmpeg.TinyImage(ctx, inputPath, outputPath,
		settings.Image.Codec, settings.Image.Scale,
		settings.Image.CRF, settings.Image.QScale)
	if err != nil {
		_ = os.Remove(outputPath) // clean up
		return "", err
	}

	// FFMPEG writes no metadata of the input file when it re-encodes an image,
	// so the metadata is transplanted into the output file. It is done before
	// the sizes are compared, since the metadata weighs in the balance.
	// A failure is not fatal: the file FFMPEG wrote is valid, it just holds no
	// metadata, which is the outcome this feature exists to avoid.
	metadataWarning := keepMetadata(outputPath, inputPath)

	outcome, err = sizeCheck(inputPath, outputPath, stats)
	if err != nil {
		_ = os.Remove(outputPath) // clean up
		return "", err
	}

	err = filetime.Copy(outputPath, inputPath)
	if err != nil {
		_ = os.Remove(outputPath) // clean up
		return outcome, err
	}

	return outcome + metadataWarning, nil
}

func doAudio(ctx context.Context, settings config.Settings,
	inputPath string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats) (
	outcome string, err error) {
	outputTempPath, outputPath := path.InputToOutput(inputPath,
		settings.OutputDirPath, settings.Audio.OutputExtension)

	outputFileExists, err := path.DoesFileExist(outputPath)
	if err != nil {
		return "", err
	}

	if outputFileExists {
		if !*settings.OverrideOutput {
			return fileAlreadyExists, nil
		}

		err = os.Remove(outputPath)
		if err != nil {
			return "", fmt.Errorf("removing existing output file: %w", err)
		}
	}

	outputDir := filepath.Dir(outputPath)
	const dirPerms os.FileMode = 0700
	err = os.MkdirAll(outputDir, dirPerms)
	if err != nil {
		return "", fmt.Errorf("cannot create parent output directory: %w", err)
	}

	defer func() {
		_ = os.Remove(outputTempPath) // clean up
	}()
	err = ffmpeg.TinyAudio(ctx, inputPath, outputTempPath,
		settings.Audio.Codec, *settings.Audio.QScale, *settings.Audio.BitRate)
	if err != nil {
		return "", err
	}

	outcome, err = sizeCheck(inputPath, outputTempPath, stats)
	if err != nil {
		return "", err
	}

	err = filetime.Copy(outputTempPath, inputPath)
	if err != nil {
		return outcome, err
	}

	err = os.Rename(outputTempPath, outputPath)
	if err != nil {
		return outcome, fmt.Errorf("renaming temp output file to final output file: %w", err)
	}

	return outcome, nil
}

func doVideo(ctx context.Context, settings config.Settings,
	inputPath string, ffmpeg *ffmpeg.FFMPEG, stats *stats.Stats,
	w io.Writer) (outcome string, err error) {
	tempOutputPath, outputPath := path.InputToOutput(inputPath,
		settings.OutputDirPath, settings.Video.OutputExtension)

	line := fmt.Sprintf("🗜️  Tinying %s ...", inputPath)
	fmt.Fprint(w, line)

	outputFileExists, err := path.DoesFileExist(outputPath)
	if err != nil {
		return "", err
	}

	if outputFileExists {
		if !*settings.OverrideOutput {
			return fileAlreadyExists, nil
		}

		err = os.Remove(outputPath)
		if err != nil {
			return "", fmt.Errorf("removing existing output file: %w", err)
		}
	}

	outputDir := filepath.Dir(outputPath)
	const dirPerms os.FileMode = 0700
	err = os.MkdirAll(outputDir, dirPerms)
	if err != nil {
		return "", fmt.Errorf("cannot create parent output directory: %w", err)
	}

	spinner.ClearLine(w)
	spinner := spinner.New(w, line, []rune{'⌛', '⏳', '🔥'})
	spinner.Start()

	defer func() {
		_ = os.Remove(tempOutputPath) // clean up
	}()
	err = ffmpeg.TinyVideo(ctx, inputPath, tempOutputPath,
		settings.Video.Scale, settings.Video.Preset, settings.Video.Codec,
		settings.Video.AudioCodec, *settings.Video.Crf)
	spinner.Stop()
	if err != nil {
		return "", err
	}

	outcome, err = sizeCheck(inputPath, tempOutputPath, stats)
	if err != nil {
		return "", err
	}

	err = filetime.Copy(tempOutputPath, inputPath)
	if err != nil {
		return outcome, err
	}

	err = os.Rename(tempOutputPath, outputPath)
	if err != nil {
		return outcome, fmt.Errorf("renaming temp output file to final output file: %w", err)
	}

	return outcome, nil
}

// keepMetadata copies the metadata of the input image file into the output one,
// and returns an empty string. It does nothing, and returns an empty string as
// well, when one of the files is in a format it cannot handle, such as an AVIF
// file written by another encoder.
//
// It never fails: the file written by the encoder is valid with or without its
// metadata, so a failure to transplant the metadata is only reported, and the
// file is kept.
func keepMetadata(outputPath, inputPath string) (warning string) {
	err := metadata.Copy(outputPath, inputPath)
	switch {
	case err == nil, errors.Is(err, metadata.ErrUnsupportedFormat):
		return ""
	default:
		return " ⚠️  Metadata not kept: " + err.Error()
	}
}

func sizeCheck(inputPath, outputPath string,
	stats *stats.Stats) (outcome string, err error) {
	inputSize, outputSize, err := size.GetSizes(inputPath, outputPath)
	if err != nil {
		return "", err
	}

	stats.InputSize += inputSize
	stats.OutputSize += outputSize

	outcome = "✔️  (" + size.DiffString(outputSize, inputSize) + ")"

	if outputSize <= inputSize {
		return outcome, nil
	}

	outcome += " 😑 Replacing output with input..."
	err = size.ReplaceBy(outputPath, inputPath)
	if err != nil {
		return outcome, err
	}
	stats.OutputSize += inputSize - outputSize
	outcome += " ✔️"
	return outcome, nil
}

func warnSignErr(err error) string {
	return " ⚠️  " + err.Error()
}

// setVideoEncoder sets in the settings the video encoder to encode with, and
// the preset of that encoder: the wanted codec when the FFMPEG build provides
// it, or the encoder tinier falls back on, which is then written as a warning.
// The wanted codec is kept, and a warning written, when the FFMPEG build
// provides none of them, in which case the video conversions will fail. A
// preset given by a user is kept as it is; the preset of the encoder is used
// otherwise, preset values being encoder specific and a codec missing from the
// build being replaced by another encoder. It does nothing when the files to
// convert hold no video.
func setVideoEncoder(ctx context.Context, settings *config.Settings,
	ffmpeg *ffmpeg.FFMPEG, videoCount int, w io.Writer) error {
	if videoCount == 0 {
		return nil
	}

	codec, found, err := ffmpeg.PickVideoCodec(ctx, settings.Video.Codec)
	if err != nil {
		return err
	}

	switch {
	case !found:
		fmt.Fprintf(w, "⚠️  No video encoder is available in this FFMPEG build, video conversions will fail.\n")
	case codec != settings.Video.Codec:
		fmt.Fprintf(w, "⚠️  The video codec %s is not available in this FFMPEG build, encoding with %s instead.\n",
			settings.Video.Codec, codec)
	}

	settings.Video.Codec = codec
	if settings.Video.Preset == "" {
		settings.Video.Preset = settings.Video.DefaultPreset()
	}

	return nil
}
