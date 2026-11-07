# Tinier

## Features

Tinier is a safe, easy to use program to make your media files tinier, with a small quality loss.

- [x] Video files (using `libsvtav1` by default)
- [x] Image files
- [x] Audio files (using `libopus` by default)

## Setup and usage

### Windows

**Compatibility**: amd64, 386, arm64

1. Download the pre-built program for your platform from the [Github releases](https://github.com/qdm12/tinier/releases).
1. Run with

    ```psh
    tinier.exe
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
    ```

### Linux

**Compatibility**: amd64, 386, armv5, armv6, armv7 and arm64

1. Download the pre-built program for your platform from the [Github releases](https://github.com/qdm12/tinier/releases).
1. Run with

    ```sh
    chmod +x tinier
    ./tinier
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
docker run -it --rm -v /your/path:/tmp/tinier qmcgaw/tinier -input-dir-path /tmp/tinier/input -output-dir-path /tmp/tinier/output
```

You can also use the following environment variables if you prefer:

| Environment variable | Default value |
| --- | --- |
| `TINIER_INPUT_DIR_PATH` | `/input` |
| `TINIER_OUTPUT_DIR_PATH` | `/output` |
| `TINIER_FFMPEG_PATH` |  |
| `TINIER_FFMPEG_MIN_VERSION` | `5.0.1` |
| `TINIER_OVERRIDE_OUTPUT` | `off` |
| `TINIER_VIDEO_SCALE` | `min(1280,iw):-2` |
| `TINIER_VIDEO_PRESET` | none, and the preset of the encoder is used: `8` for `libsvtav1`, `fast` for `libx264`, `libx265` and `libaom-av1` |
| `TINIER_VIDEO_CODEC` | `libsvtav1` |
| `TINIER_VIDEO_AUDIO_CODEC` | `libopus` |
| `TINIER_VIDEO_OUTPUT_EXTENSION` | `.mp4` |
| `TINIER_VIDEO_EXTENSIONS` | `.mp4,.mov,.avi` |
| `TINIER_VIDEO_SKIP` | `no` |
| `TINIER_VIDEO_CRF` | `23` |
| `TINIER_IMAGE_SCALE` | `min(1280,iw):-1` |
| `TINIER_IMAGE_OUTPUT_EXTENSION` | `.jpg` |
| `TINIER_IMAGE_EXTENSIONS` | `.jpg,.jpeg,.png` |
| `TINIER_IMAGE_SKIP` | `no` |
| `TINIER_IMAGE_CODEC` | `mjpeg` |
| `TINIER_IMAGE_QSCALE` | `5` |
| `TINIER_IMAGE_CRF` | `35` |
| `TINIER_AUDIO_CODEC` | `libopus` |
| `TINIER_AUDIO_OUTPUT_EXTENSION` | `.opus` |
| `TINIER_AUDIO_EXTENSIONS` | `.mp3,.flac` |
| `TINIER_AUDIO_SKIP` | `no` |
| `TINIER_AUDIO_QSCALE` | `5` |
| `TINIER_AUDIO_BITRATE` | `32k` |

## General usage

```sh
tinier -help
Usage of tinier:
  -audio-bitrate string
        Audio ffmpeg codec bitrate string value. (default "32k")
  -audio-codec string
        Audio ffmpeg codec. (default "libopus")
  -audio-extensions string
        CSV list of audio file extensions. (default ".mp3,.flac")
  -audio-output-extension string
        Audio output file extension to use. (default ".opus")
  -audio-qscale int
        Audio ffmpeg QScale value. (default 5)
  -audio-skip
        Skip audio files.
  -ffmpeg-minversion string
        FFMPEG binary minimum version requirement. (default "5.0.1")
  -ffmpeg-path string
        FFMPEG binary path.
  -image-codec string
        Image ffmpeg codec. (default "mjpeg")
  -image-crf int
        Image ffmpeg crf value, only used by the libaom-av1 codec. (default 35)
  -image-extensions string
        CSV list of image file extensions. (default ".jpg,.jpeg,.png")
  -image-output-extension string
        Image output file extension to use. (default ".jpg")
  -image-qscale int
        Image ffmpeg qscale:v value, only used by the mjpeg codec. (default 5)
  -image-scale string
        Image ffmpeg scale value. (default "min(1280,iw):-1")
  -image-skip
        Skip image files.
  -input-dir-path string
        Input directory path. (default "input")
  -output-dir-path string
        Output directory path. (default "output")
  -override
        Override files in the output directory.
  -video-codec string
        Video ffmpeg codec. (default "libsvtav1")
  -video-audio-codec string
        Video audio ffmpeg codec. (default "libopus")
  -video-crf int
        Video ffmpeg CRF value. (default 23)
  -video-extensions string
        CSV list of video file extensions. (default ".mp4,.mov,.avi")
  -video-output-extension string
        Video output file extension to use. (default ".mp4")
  -video-preset string
        Video ffmpeg preset. Defaults to the preset of the encoder: 8 for libsvtav1, fast for libx264, libx265 and libaom-av1.
  -video-scale string
        Video ffmpeg scale value. (default "min(1280,iw):-2")
  -video-skip
        Skip video files.
```

## Implementation details

### Ffmpeg detection

`tinier` manages its own dependency `ffmpeg` by:

1. looking at the user given ffmpeg path
1. looking at any `ffmpeg` in the system path
1. falling back to downloading a static ffmpeg build for your platform

In all cases it skips a certain `ffmpeg` if it doesn't match the default minimum version `5.0.1`, which can be changed with `-ffmpeg-minversion`.

The encoders of the chosen build are read too: a video encoder the build does not provide is replaced by `libx265`, then by `libx264`, and the replacement is reported. The `libaom-av1` encoder is never picked as a replacement, being an order of magnitude slower than the others. The preset of the encoder follows the same path, preset values being encoder specific: a preset you give is kept as it is, and an unknown one is refused before any conversion.

### Metadata

FFMPEG writes no metadata when it re-encodes an image, which is why `tinier` copies the metadata of the input file into the output file itself. It is transplanted as it is, never re-serialized, so information FFMPEG cannot express survives: maker notes, GPS coordinates, embedded thumbnails, XMP, ICC profiles and IPTC data.

Two parts of the EXIF data describe the pixels, and FFMPEG changes them:

- the rotation of the image is applied to the pixels, so the orientation tag is reset to normal
- the dimension tags are updated to the dimensions of the new pixels

The metadata of a JPEG, PNG or AVIF input file is copied, and written to a JPEG output file, which `-image-output-extension` defaults to. The metadata of a file of another format, such as HEIC, is not read, and no metadata is written to a non-JPEG output file.

Only the metadata which can be made consistent with the new pixels is copied: EXIF, XMP, ICC profile, IPTC and the comment. The remaining application specific segments of a JPEG input file (`APP14` for Adobe, `APP6` for MPF, ...) and the chunks of a PNG input file which a JPEG file has no equivalent of are left out: the transparency and pixel density chunks, and the text chunks whose keyword means nothing in a JPEG file, such as `Title`, `Author` or `Description`. The contents of a JPEG segment can also point at positions of the file which the new pixels no longer match.

### Safety

- `tinier` can **be stopped at anytime** and pick up again safely
- `tinier` copies over all files from the input directory to the output directory, even if untouched.
- `tinier` encodes videos to a temporary directory and only moves them to the output directory when completed.
- `tinier` does not delete any file from the input directory

## Limitations

- AVIF files are left untouched by default: converting one to JPEG loses the transparency and the high dynamic range it can hold, and the conversion is most of the time bigger than the file it converts. Add the `.avif` extension to `-image-extensions` to convert them anyway, metadata included
- the metadata of a HEIC input image is not preserved, and converting one to JPEG loses the high dynamic range it can hold, so converting this format is best avoided
- a motion photo, taken by most Android phones, holds a JPEG photo and an MP4 video in the same file: only the photo is kept, as a single JPEG still image, and the video part is lost
- FFMPEG keeps one track of each kind, so the extra audio tracks of a video, and the picture attached to an audio file, are not kept
- application specific segments of input JPEG files, and chunks of input PNG files which have no JPEG equivalent, are not copied, except EXIF, XMP, ICC profile, IPTC and the comment
- file creation time (OS dependent) is not preserved
