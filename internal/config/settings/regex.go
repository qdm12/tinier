package settings

import "regexp"

var (
	regexExtension = regexp.MustCompile(`^\.[a-z0-9]{1,5}$`)
	regexScale     = regexp.MustCompile(`^([0-9]+|-1):([0-9]+|-1)`)
)
