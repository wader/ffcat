package linebuffer

import (
	"bytes"
	"strings"
)

// Fn calls function for each line written
type Fn struct {
	buf bytes.Buffer
	fn  func(line string)
}

// NewFn create new buffer that calls function foreach line written
func NewFn(fn func(line string)) *Fn {
	return &Fn{fn: fn}
}

func (fn *Fn) Write(p []byte) (n int, err error) {
	fn.buf.Write(p)
	b := fn.buf.Bytes()
	pos := 0

	for {
		i := bytes.IndexAny(b[pos:], "\n\r")
		if i < 0 {
			break
		}

		fn.fn(string(b[pos : pos+i+1]))
		pos += i + 1
	}
	fn.buf.Reset()
	fn.buf.Write(b[pos:])

	return len(p), nil
}

// Close flushes any data left in the buffer as a line
func (fn *Fn) Close() error {
	if fn.buf.Len() > 0 {
		fn.fn(string(fn.buf.Bytes()))
	}
	fn.buf.Reset()
	return nil
}

// LastLines buffers the last n lines
type LastLines struct {
	Fn
	current int
	lines   []string
}

// NewLastLines creates a new limited line buffer that buffers the last n lines
func NewLastLines(limit int) *LastLines {
	ll := &LastLines{
		current: 0,
		lines:   make([]string, limit),
	}
	ll.fn = ll.addLine
	return ll
}

func (lb *LastLines) addLine(line string) {
	lb.lines[lb.current] = line
	lb.current = (lb.current + 1) % len(lb.lines)
}

// String returns last n lines as a string
func (lb *LastLines) String() string {
	var ls []string
	for i := 0; i < len(lb.lines); i++ {
		ls = append(ls, lb.lines[(lb.current+i)%len(lb.lines)])
	}
	return strings.Join(ls, "")
}

// Bytes returns last n lines as bytes
func (lb *LastLines) Bytes() []byte {
	return []byte(lb.String())
}
