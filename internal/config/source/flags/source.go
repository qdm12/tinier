package flags

import (
	"github.com/qdm12/govalid"
	"github.com/qdm12/govalid/separated"
)

type Source struct {
	args      []string
	validator Validator
}

type Validator interface {
	ValidateSeparated(value string, options ...separated.Option) (slice []string, err error)
}

func New(args []string) *Source {
	return &Source{
		args:      args,
		validator: govalid.New(),
	}
}

func (s *Source) String() string {
	return "command line flags"
}
