package config

import "regexp"

const (
	// scaleDimension is one dimension of a scale, in the syntax of FFMPEG:
	// a number, one of the -1 and -2 values asking FFMPEG to compute it
	// from the other dimension, or an expression capping it to the value
	// of the input file, such as min(1280,iw). The -2 value is used
	// instead of -1 when the encoder requires even dimensions.
	scaleDimension = `([0-9]+|-1|-2|(min|max)\((iw|ih|[0-9]+),(iw|ih|[0-9]+)\))`
	// scaleSource is a scale, made of two dimensions separated by a
	// colon, like 1280:-1 or min(1280,iw):-2.
	scaleSource = `^` + scaleDimension + `:` + scaleDimension + `$`
)

var (
	regexExtension = regexp.MustCompile(`^\.[a-z0-9]{1,5}$`)
	regexScale     = regexp.MustCompile(scaleSource)
)
