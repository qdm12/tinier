package stats

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/qdm12/tinier/internal/size"
)

type Stats struct {
	Failures   int
	InputSize  int64
	OutputSize int64
	Start      time.Time
}

func New() *Stats {
	return &Stats{
		Start: time.Now(),
	}
}

func (s *Stats) Finish(w io.Writer) {
	if s.InputSize == 0 {
		return
	}

	var parts []string
	switch s.Failures {
	case 0:
	case 1:
		parts = append(parts, "😬 encountered a single failed conversion")
	default:
		parts = append(parts, "😬 encountered "+fmt.Sprint(s.Failures)+" failed conversions")
	}

	parts = append(parts, size.DiffString(s.OutputSize, s.InputSize))
	parts = append(parts, fmt.Sprintf("took %s", time.Since(s.Start).Round(time.Second)))

	fmt.Fprintln(w, "Finished: "+strings.Join(parts, " | "))
}
