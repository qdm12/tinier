package cmd

var _ Runner = (*Cmd)(nil)

// Cmd handles running subprograms synchronously or asynchronously.
type Cmd struct{}

func New() *Cmd {
	return &Cmd{}
}
