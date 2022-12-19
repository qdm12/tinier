package semver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func MustParse(s string) (version Semver) {
	version, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return version
}

var (
	ErrSemverNotFound = errors.New("semver not found")
)

var (
	regexFullSemver    = regexp.MustCompile(`[1-9][0-9]*\.[0-9]+\.[0-9]+`)
	regexPartialSemver = regexp.MustCompile(`[1-9][0-9]*\.[0-9]+`)
)

func Extract(s string) (version Semver, err error) {
	var extractedSemverString string
	extractedSemverString = regexFullSemver.FindString(s)
	if extractedSemverString == "" {
		extractedSemverString = regexPartialSemver.FindString(s)
		if extractedSemverString == "" {
			return version, fmt.Errorf("%w: in %q", ErrSemverNotFound, s)
		}
	}

	return Parse(extractedSemverString)
}

var (
	ErrNumberOfFields = errors.New("semver has the wrong number of fields")
)

func Parse(s string) (version Semver, err error) {
	parts := strings.Split(s, ".")
	const (
		fullSemverFields    = 3
		partialSemverFields = 2
	)
	switch len(parts) {
	case fullSemverFields, partialSemverFields:
	default:
		return version, fmt.Errorf("%w: %s has %d fields but expected %d or %d fields",
			ErrNumberOfFields, s, len(parts), fullSemverFields, partialSemverFields)
	}

	major, err := parseUint(parts[0])
	if err != nil {
		return version, fmt.Errorf("parsing major version: %w", err)
	}

	minor, err := parseUint(parts[1])
	if err != nil {
		return version, fmt.Errorf("parsing minor version: %w", err)
	}

	var patch uint
	if len(parts) == fullSemverFields {
		patch, err = parseUint(parts[2])
		if err != nil {
			return version, fmt.Errorf("parsing patch version: %w", err)
		}
	}

	return Semver{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func parseUint(s string) (n uint, err error) {
	nUint64, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(nUint64), nil
}
