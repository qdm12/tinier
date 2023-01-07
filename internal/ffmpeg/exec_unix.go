//go:build !windows

package ffmpeg

import (
	"os/exec"
	"syscall"
)

func patchCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
