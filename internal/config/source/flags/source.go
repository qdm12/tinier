package flags

type Source struct {
	args []string
}

func New(args []string) *Source {
	return &Source{
		args: args,
	}
}

func (s *Source) String() string {
	return "command line flags"
}
