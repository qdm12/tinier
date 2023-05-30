package env

type Source struct{}

func New() *Source {
	return &Source{}
}

func (s *Source) String() string {
	return "environment variables"
}
