package source

import (
	"github.com/qdm12/tinier/internal/config/settings"
)

type Source interface {
	String() string
	Read() (s settings.Settings, err error)
}
