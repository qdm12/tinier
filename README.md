# Tinier

## Features

Tinier is a safe, easy to use program to make your media files tinier, with a small quality loss.

- [x] Video files (using `libaom-av1` by default)
- [x] Image files
- [x] Audio files (using `libmp3lame` by default)

## Setup and usage

### Windows

**Compatibility**: amd64, 386, arm64

1. Download the pre-built program for your platform from the [Github releases](https://github.com/qdm12/tinier/releases).
1. Run with

    ```psh
    tinier.exe
    # Show list of flag options
    tinier.exe -help
    ```

1. 💁 If your CPU is `386` or `arm64`, you need to install [`ffmpeg`](https://ffmpeg.org/) manually.

### Mac OSX

**Compatibility**: amd64, arm64

1. Install [`ffmpeg`](https://ffmpeg.org/), usually `brew install ffmpeg` does it.
1. Download the pre-built program for your platform from the [Github releases](https://github.com/qdm12/tinier/releases).
1. Run with

    ```zsh
    chmod +x tinier
    ./tinier
    # Show list of flag options
    ./tinier.exe -help
    ```

### Linux

**Compatibility**: amd64, 386, armv5, armv6, armv7 and arm64

1. Download the pre-built program for your platform from the [Github releases](https://github.com/qdm12/tinier/releases).
1. Run with

    ```sh
    chmod +x tinier
    ./tinier
    # Show list of flag options
    ./tinier.exe -help
    ```

### From source

1. Install [go](https://go.dev/)
1. Install tinier from source:

    ```sh
    go install github.com/qdm12/tinier/cmd/tinier
    ```

1. 💁 Depending on your platform, you might have to install [`ffmpeg`](https://ffmpeg.org/) manually.

### Docker

**Compatibility**: x86_64, x86, aarch64, armhf, armv7, ppc64le and s390x

```sh
docker run -it --rm -v /your/path:/tmp/tinier qmcgaw/tinier -input /tmp/tinier/input -output /tmp/tinier/output
```

You can also use the following environment variables if you prefer:

| Environment variable | Default value |
| --- | --- |
| `TINIER_INPUT_DIR_PATH` | `/input` |
| `TINIER_OUTPUT_DIR_PATH` | `/output` |
| `TINIER_FFMPEG_PATH` |  |
| `TINIER_FFMPEG_MIN_VERSION` | `5.0.1` |
| `TINIER_OVERRIDE_OUTPUT` | `off` |
| `TINIER_VIDEO_SCALE` | `1280:-1` |
| `TINIER_VIDEO_PRESET` | `medium` |
| `TINIER_VIDEO_CODEC` | `libaom-av1` |
| `TINIER_VIDEO_OUTPUT_EXTENSION` | `.mp4` |
| `TINIER_VIDEO_EXTENSIONS` | `.mp4,.mov,.avi` |
| `TINIER_VIDEO_SKIP` | `no` |
| `TINIER_VIDEO_CRF` | `23` |
| `TINIER_IMAGE_SCALE` | `5` |
| `TINIER_IMAGE_OUTPUT_EXTENSION` | `.jpg` |
| `TINIER_IMAGE_EXTENSIONS` | `.jpg,.jpeg` |
| `TINIER_IMAGE_SKIP` | `no` |
| `TINIER_IMAGE_QSCALE` | `5` |
| `TINIER_AUDIO_CODEC` | `libmp3lame` |
| `TINIER_AUDIO_OUTPUT_EXTENSION` | `.mp3` |
| `TINIER_AUDIO_EXTENSIONS` | `.mp3,.flac` |
| `TINIER_AUDIO_SKIP` | `no` |
| `TINIER_AUDIO_QSCALE` | `5` |

## General usage

```sh
tinier -help
Usage of /tmp/go-build4139756528/b001/exe/main:
  -audiocodec string
        Audio ffmpeg codec. (default "libmp3lame")
  -audioextensions string
        CSV list of audio file extensions. (default ".mp3,.flac")
  -audiooutputextension string
        Audio output file extension to use. (default ".mp3")
  -audioqscale int
        Audio ffmpeg QScale value. (default 5)
  -audioskip
        Skip audio files.
  -ffmpegminversion string
        FFMPEG binary minimum version requirement. (default "5.0.1")
  -ffmpegpath string
        FFMPEG binary path.
  -imageextensions string
        CSV list of image file extensions. (default ".jpg,.jpeg")
  -imageoutputextension string
        Image output file extension to use. (default ".jpg")
  -imageqscale int
        Image ffmpeg qscale:v value. (default 5)
  -imagescale string
        Image ffmpeg scale value. (default "1280:-1")
  -imageskip
        Skip image files.
  -inputdirpath string
        Input directory path. (default "input")
  -outputdirpath string
        Output directory path. (default "output")
  -override
        Override files in the output directory.
  -videocodec string
        Video ffmpeg codec. (default "libaom-av1")
  -videocrf int
        Video ffmpeg CRF value. (default 23)
  -videoextensions string
        CSV list of video file extensions. (default ".mp4,.mov,.avi")
  -videooutputextension string
        Video output file extension to use. (default ".mp4")
  -videopreset string
        Video ffmpeg preset. (default "medium")
  -videoscale string
        Video ffmpeg scale value. (default "1280:-1")
  -videoskip
        Skip video files.
```

## Implementation details

### Ffmpeg detection

`tinier` manages its own dependency `ffmpeg` by:

1. looking at the user given ffmpeg path
1. looking at any `ffmpeg` in the system path
1. falling back to downloading a static ffmpeg build for your platform

In all cases it skips a certain `ffmpeg` if it doesn't match the default minimum version `5.0.1`, which can be changed with `-ffmpegminversion`.

### Safety

- `tinier` can **be stopped at anytime** and pick up again safely
- `tinier` copies over all files from the input directory to the output directory, even if untouched.
- `tinier` encodes videos to a temporary directory and only moves them to the output directory when completed.
- `tinier` does not delete any file from the input directory

## Limitations

- EXIF data is not preserved
- file creation time (OS dependent) is not preserved
