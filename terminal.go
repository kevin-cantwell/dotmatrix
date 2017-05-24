package dotmatrix

import (
	"fmt"
	"io"
)

type Terminal interface {
	ResetCursor(rows int)
	ShowCursor(show bool)
} // Reset Text Color: t.writer().Write([]byte("\033[0m"))

type Xterm struct {
	Writer io.Writer
}

// Move the cursor to the beginning of the line and up rows
func (term *Xterm) ResetCursor(rows int) {
	term.Writer.Write([]byte(fmt.Sprintf("\033[999D\033[%dA", rows)))
}

func (term *Xterm) ShowCursor(show bool) {
	if show {
		term.Writer.Write([]byte("\033[?12l\033[?25h"))
	} else {
		term.Writer.Write([]byte("\033[?25l"))
	}
}
