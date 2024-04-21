package esl

import (
	"bufio"
	"io"
	"strings"
)

// writeTo writes to the given io.Writer using a provided function.
//
// It takes a writer `w` of type io.Writer and a function `f` as parameters.
// The function `f` is executed with a bufio.Writer as its argument.
//
// It returns the number of bytes written and an error if any.
func writeTo(w io.Writer, f func(w *bufio.Writer)) (int64, error) {
	buf := bufio.NewWriter(w) // initialize buffered writer
	nn := buf.Buffered()      // store current buffered length
	f(buf)                    // execute function

	return int64(buf.Buffered() - nn), buf.Flush() // write buffered content
}

// wstr concatenates the string representation of the io.WriterTo interface
// parameter into a single string.
//
// It takes an io.WriterTo parameter, which is an interface type that can be
// written to an io.Writer. The function writes the string representation of the
// parameter to a strings.Builder and returns the resulting string.
//
// The function does not handle any errors that may occur during the writing
// process.
//
// It returns a string that contains the concatenated string representation of
// the io.WriterTo parameter.
func wstr(w io.WriterTo) string {
	var b strings.Builder

	w.WriteTo(&b) //nolint:errcheck // write to buffer

	return b.String()
}

// trimLeft removes leading spaces and tabs from the given byte slice and returns the result as a string.
func trimLeft(b []byte) string {
	for i := range len(b) {
		if b[i] != ' ' && b[i] != '\t' {
			return string(b[i:])
		}
	}

	return ""
}
