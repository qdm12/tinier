package mux

import (
	"fmt"

	"github.com/qdm12/tinier/internal/config/settings"
	"github.com/qdm12/tinier/internal/config/source"
)

var _ source.Source = (*Source)(nil)

type Source struct {
	sources []source.Source
}

func New(sources ...source.Source) *Source {
	return &Source{
		sources: sources,
	}
}

func (s *Source) String() string {
	return "mux"
}

func (s *Source) Read() (settings settings.Settings, err error) {
	for _, source := range s.sources {
		newSettings, err := source.Read()
		if err != nil {
			return settings, fmt.Errorf("%s source: %w", source, err)
		}
		settings.MergeWith(newSettings)
	}

	return settings, nil
}
