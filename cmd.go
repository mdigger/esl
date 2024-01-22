package esl

import (
	"bufio"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

type command struct {
	name    string
	params  string
	jobUUID string
	headers map[string]string
	body    string
}

// cmd creates a new command with the given name and parameters.
func cmd(name string, params ...string) command {
	if name == "" {
		panic("command name cannot be empty")
	}

	return command{
		name:   name,
		params: strings.Join(params, " "),
	}
}

// WithJobUUID sets the jobUUID field of the command struct.
func (c command) WithJobUUID(id string) command {
	c.jobUUID = strings.TrimSpace(id)
	return c
}

// WithMessage sets the headers and body of the command.
func (c command) WithMessage(h map[string]string, body string) command {
	c.headers = h
	c.body = body
	return c
}

// WriteTo writes the command to the given writer.
func (c command) WriteTo(w io.Writer) (n int64, err error) {
	//nolint:errcheck // writing to buffer
	return writeTo(w, func(w *bufio.Writer) {
		w.WriteString(c.name)
		if c.params != "" {
			w.WriteByte(' ')
			w.WriteString(c.params)
		}
		if c.jobUUID != "" {
			w.WriteString("\nJob-UUID: ")
			w.WriteString(c.jobUUID)
		}
		if len(c.headers) > 0 {
			keys := make([]string, 0, len(c.headers))
			for k := range c.headers {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				w.WriteByte('\n')
				w.WriteString(k)
				w.WriteString(": ")
				w.WriteString(c.headers[k])
			}
		}
		if c.body != "" {
			w.WriteString("\ncontent-length: ")
			w.WriteString(strconv.Itoa(len(c.body)))
			w.WriteString("\n\n")
			w.WriteString(c.body)
		}
	})
}

// String returns the string representation of the command.
func (c command) String() string {
	return wstr(c)
}

// LogValue returns a slog.Value object representing the command.
func (c command) LogValue() slog.Value {
	attr := make([]slog.Attr, 0, 3)
	attr = append(attr, slog.String("name", c.name))
	if c.params != "" {
		if c.name == "auth" {
			c.params = "*****" // hide password
		}
		attr = append(attr, slog.String("params", c.params))
	}
	if c.jobUUID != "" {
		attr = append(attr, slog.String("job-uuid", c.jobUUID))
	}

	return slog.GroupValue(attr...)
}

// IsZero checks if the command is zero.
func (c command) IsZero() bool {
	return c.name == ""
}
