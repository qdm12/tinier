package semver

import "fmt"

type Semver struct {
	Major uint
	Minor uint
	Patch uint
}

func (s Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}

func (s *Semver) Equal(other Semver) bool {
	return s.Major == other.Major &&
		s.Minor == other.Minor &&
		s.Patch == other.Patch
}

func (s *Semver) Before(other Semver) bool {
	if s.Major < other.Major {
		return true
	}

	if s.Major == other.Major && s.Minor < other.Minor {
		return true
	}

	if s.Major == other.Major && s.Minor == other.Minor && s.Patch < other.Patch {
		return true
	}

	return false
}

func (s *Semver) After(other Semver) bool {
	return s.Major > other.Major ||
		s.Minor > other.Minor ||
		s.Patch > other.Patch
}
