package cmd

import (
	"strings"
)

type Runner interface {
	Run(cmd ExecCmd) (output string, err error)
}

// Run runs a command in a blocking manner, returning its output and
// an error if it failed.
func (c *Cmd) Run(cmd ExecCmd) (output string, err error) {
	stdout, err := cmd.CombinedOutput()
	output = string(stdout)
	output = strings.TrimSuffix(output, "\n")
	lines := stringToLines(output)
	output = strings.Join(lines, "\n")
	return output, err
}

func stringToLines(s string) (lines []string) {
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}
