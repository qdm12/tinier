package env

import (
	"github.com/qdm12/govalid"
	"github.com/qdm12/govalid/binary"
	"github.com/qdm12/govalid/integer"
	"github.com/qdm12/govalid/separated"
)

type Source struct {
	validator Validator
}

type Validator interface {
	ValidateBinary(value string, options ...binary.Option) (enabled bool, err error)
	ValidateInteger(value string, options ...integer.Option) (integer int, err error)
	ValidateSeparated(value string, options ...separated.Option) (slice []string, err error)
}

func New() *Source {
	return &Source{
		validator: govalid.New(),
	}
}

func (s *Source) String() string {
	return "environment variables"
}
