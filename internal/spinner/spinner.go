package spinner

import (
	"fmt"
	"io"
	"time"
)

type Spinner struct {
	// Parameters
	writer   io.Writer
	line     string
	sequence []rune
	period   time.Duration
	// Internals
	stop    chan struct{}
	stopped chan struct{}
}

func New(w io.Writer, line string, sequence []rune) *Spinner {
	stop := make(chan struct{})
	stopped := make(chan struct{})
	return &Spinner{
		writer:   w,
		line:     line,
		sequence: sequence,
		period:   time.Second,
		stop:     stop,
		stopped:  stopped,
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.stopped)

		ticker := time.NewTicker(s.period)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.stop:
				ClearLine(s.writer)
				fmt.Fprint(s.writer, s.line)
				return
			case <-ticker.C:
				ClearLine(s.writer)
				fmt.Fprint(s.writer, s.line+string(s.sequence[i]))
				i++
				if i == len(s.sequence) {
					i = 0
				}
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.stopped
}
