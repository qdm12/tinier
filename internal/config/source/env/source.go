package env

import (
	"os"

	"github.com/qdm12/gosettings/sources/env"
)

type Source struct {
	env *env.Env
}

func New() *Source {
	return &Source{
		env: env.New(os.Environ(), nil),
	}
}

func (s *Source) String() string {
	return "environment variables"
}
