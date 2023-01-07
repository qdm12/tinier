//go:build windows

package ffmpeg

import (
	"os/exec"
)

func patchCmd(cmd *exec.Cmd) {}
