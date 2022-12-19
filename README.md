# Tinier

🚧 Work in progress, do not use yet 🚧

## Features

Tinier is here to make your media files tinier, with a small quality loss.

- [x] Video files
- [x] Image files
- [x] Audio files

It manages its own dependency (`ffmpeg`) by:

1. looking at the user given ffmpeg path
1. looking at any `ffmpeg` in the system path
1. falling back to downloading a static ffmpeg build for your platform

### Compatibility

#### Binary program

The Go program is pre-built for the following platforms:

- Linux: amd64, 386, armv5, armv6, armv7 and arm64
- OSX: amd64, arm64
- Windows: amd64, 386, arm64
- 💁 [Create an issue if you need another platform](https://github.com/qdm12/tinier/issues/new) it should be easy to add.

If your platform is listed below, `tinier` can take care of downloading the `ffmpeg` program from you:

- Linux: amd64, arm64, 386, armv7, armv6, armv5
- Windows: amd64

Otherwise you can install [`ffmpeg`](https://ffmpeg.org/) yourself on your platform and specify it with `-ffmpegpath`.

#### Docker image

The Docker image is based on Alpine and is compatible with the following CPU architectures:
x86_64, x86, aarch64, armhf, armv7, ppc64le and s390x

## Usage

```sh
tinier -inputdirpath ./yourinputdir -outputdirpath ./youroutputdir
```

## Installation

- Download from [Github releases](https://github.com/qdm12/tinier/releases) for your platform (work in progress)
- Build from source

  ```sh
  go install github.com/qdm12/tinier/cmd/tinier
  ```

- Use the Docker image (work in progress)

    ```sh
    docker run -it --rm -v /your/path:/tmp/tinier qmcgaw/tinier -input /tmp/tinier/input -output /tmp/tinier/output
    ```

    You can also use environment variables listed in the [Dockerfile](Dockerfile#58)

## Limitations

For now:

- EXIF data is not preserved
- file creation time (OS dependent) is not preserved

## TODOs

1. Settings `.ToLines` method using `qdm12/gotree` and `.String()`
1. Read names with case sensitivity for their extension
1. Keep file creation time (OS dependent)
1. Keep EXIF data
1. Bundle ffmpeg in Docker image with <https://github.com/jrottenberg/ffmpeg/blob/main/docker-images/5.0/scratch313/Dockerfile> or using static builds?
