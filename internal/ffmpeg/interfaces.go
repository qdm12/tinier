package ffmpeg

import (
	"net/http"

	"github.com/qdm12/tinier/internal/cmd"
)

type Runner interface {
	Run(cmd cmd.ExecCmd) (output string, err error)
}

type HTTPClient interface {
	Do(request *http.Request) (response *http.Response, err error)
}
