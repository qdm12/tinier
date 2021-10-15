package spinner

import (
	"fmt"
	"io"
)

func ClearLine(w io.Writer) {
	fmt.Fprint(w, "\033[2K")
	fmt.Fprintln(w)
	fmt.Fprint(w, "\033[1A")
}
