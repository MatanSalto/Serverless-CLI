package utils

import "io"

// ANSI codes for coloring output
const (
	ansiDimCyan = "\033[2;36m"
	ansiReset   = "\033[0m"
)

// ColorWriter wraps an io.Writer and applies an ANSI color prefix on first write,
// then reset on Reset(). Used so job logs can appear in a different color than CLI messages.
type ColorWriter struct {
	w      io.Writer
	prefix string
	reset  string
	done   bool
}

// NewDimCyanWriter returns a ColorWriter that writes to the io writer in dim cyan.
func NewDimCyanWriter(w io.Writer) *ColorWriter {
	return &ColorWriter{w: w, prefix: ansiDimCyan, reset: ansiReset}
}

func (c *ColorWriter) Write(p []byte) (n int, err error) {
	if !c.done && len(p) > 0 {
		if _, err := c.w.Write([]byte(c.prefix)); err != nil {
			return 0, err
		}
		c.done = true
	}
	return c.w.Write(p)
}

// Reset writes the ANSI reset code so the terminal returns to default styling.
func (c *ColorWriter) Reset() {
	if c.done {
		_, _ = c.w.Write([]byte(c.reset))
		c.done = false
	}
}
