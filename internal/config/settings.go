package config

import (
	"fmt"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
	"github.com/qdm12/tinier/internal/semver"
)

type Settings struct {
	InputDirPath     string
	OutputDirPath    string
	FfmpegPath       *string
	FfmpegMinVersion string
	OverrideOutput   *bool
	Video            Video
	Image            Image
	Audio            Audio
	Log              Log
}

// OverrideWith sets fields in the receiving settings
// from non-zero fields from the other settings given.
func (s *Settings) OverrideWith(other Settings) {
	s.InputDirPath = gosettings.OverrideWithComparable(s.InputDirPath, other.InputDirPath)
	s.OutputDirPath = gosettings.OverrideWithComparable(s.OutputDirPath, other.OutputDirPath)
	s.FfmpegPath = gosettings.OverrideWithPointer(s.FfmpegPath, other.FfmpegPath)
	s.FfmpegMinVersion = gosettings.OverrideWithComparable(s.FfmpegMinVersion, other.FfmpegMinVersion)
	s.OverrideOutput = gosettings.OverrideWithPointer(s.OverrideOutput, other.OverrideOutput)
	s.Video.overrideWith(other.Video)
	s.Image.overrideWith(other.Image)
	s.Audio.overrideWith(other.Audio)
	s.Log.overrideWith(other.Log)
}

// SetDefaults sets the defaults to all the zero-ed fields
// in the receiving settings.
func (s *Settings) SetDefaults() {
	s.InputDirPath = gosettings.DefaultComparable(s.InputDirPath, "input")
	s.OutputDirPath = gosettings.DefaultComparable(s.OutputDirPath, "output")
	s.FfmpegPath = gosettings.DefaultPointer(s.FfmpegPath, "")
	s.FfmpegMinVersion = gosettings.DefaultComparable(s.FfmpegMinVersion, "5.0.1")
	s.OverrideOutput = gosettings.DefaultPointer(s.OverrideOutput, false)
	s.Video.setDefaults()
	s.Image.setDefaults()
	s.Audio.setDefaults()
	s.Log.setDefaults()
}

// Validate validates all the settings are correct.
// Note `.SetDefaults()` must be called to ensure all
// the fields are not their zeroed value such as `nil`.
func (s *Settings) Validate() (err error) {
	_, err = os.Stat(s.InputDirPath)
	if err != nil {
		return fmt.Errorf("input directory: %w", err)
	}

	if *s.FfmpegPath != "" {
		_, err = os.Stat(*s.FfmpegPath)
		if err != nil {
			return fmt.Errorf("ffmpeg path: %w", err)
		}
	}

	if s.FfmpegMinVersion != "" {
		_, err = semver.Parse(s.FfmpegMinVersion)
		if err != nil {
			return fmt.Errorf("minimum ffmpeg version: %w", err)
		}
	}

	mapping := map[string]func() (err error){
		"video": s.Video.validate,
		"image": s.Image.validate,
		"audio": s.Audio.validate,
		"log":   s.Log.validate,
	}

	for name, validate := range mapping {
		err = validate()
		if err != nil {
			return fmt.Errorf("%s settings: %w", name, err)
		}
	}

	return nil
}

// toLinesNode returns a gotree.Node with the settings
// as a formatted tree node.
func (s *Settings) toLinesNode() *gotree.Node {
	node := gotree.New("Settings:")
	node.Appendf("Input directory: %s", s.InputDirPath)
	node.Appendf("Output directory: %s", s.OutputDirPath)
	if *s.FfmpegPath != "" {
		node.Appendf("FFMPEG path: %s", *s.FfmpegPath)
	}
	node.Appendf("FFMPEG minimum version: %s", s.FfmpegMinVersion)
	node.Appendf("Override existing output: %s", yesno(*s.OverrideOutput))
	node.AppendNode(s.Video.toLinesNode())
	node.AppendNode(s.Image.toLinesNode())
	node.AppendNode(s.Audio.toLinesNode())
	node.AppendNode(s.Log.toLinesNode())
	return node
}

func (s Settings) String() string {
	return s.toLinesNode().String()
}

func (s *Settings) Read(reader *reader.Reader) (err error) {
	s.InputDirPath = reader.String("INPUT_DIR_PATH")
	s.OutputDirPath = reader.String("OUTPUT_DIR_PATH")
	s.FfmpegPath = reader.Get("FFMPEG_PATH")
	s.FfmpegMinVersion = reader.String("FFMPEG_MIN_VERSION")

	s.OverrideOutput, err = reader.BoolPtr("OVERRIDE_OUTPUT")
	if err != nil {
		return err
	}

	err = s.Image.read(reader)
	if err != nil {
		return fmt.Errorf("image settings: %w", err)
	}

	err = s.Video.read(reader)
	if err != nil {
		return fmt.Errorf("video settings: %w", err)
	}

	err = s.Audio.read(reader)
	if err != nil {
		return fmt.Errorf("audio settings: %w", err)
	}

	s.Log.read(reader)
	return nil
}
