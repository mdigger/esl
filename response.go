package esl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
)

type response struct {
	contentType string // Content-Type
	text        string // Reply-Text
	jobUUID     string // Job-UUID
	body        []byte // Body
}

// ContentType returns the content type of the response.
func (r response) ContentType() string {
	return r.contentType
}

// Text returns the text value of the response.
func (r response) Text() string {
	return r.text
}

// JobUUID returns the job UUID of the response.
func (r response) JobUUID() string {
	return r.jobUUID
}

// ContentLength returns the length of the response body in bytes.
func (r response) ContentLength() int {
	return len(r.body)
}

// Body returns the body of the response as a string.
func (r response) Body() string {
	return string(r.body)
}

// AsErr checks the content type of the response and returns an error if it matches a specific case.
func (r response) AsErr() error {
	switch r.contentType {
	case disconnectNotice:
		return io.EOF
	case commandReply:
		if strings.HasPrefix(r.text, "-ERR") {
			return errors.New(r.text)
		}
	case "api/response":
		if bytes.HasPrefix(r.body, []byte("-ERR")) {
			return errors.New(string(r.body))
		}
	}

	return nil
}

// WriteTo writes the response to the provided io.Writer.
//
// It writes the response headers to the writer, including the Content-Type,
// Reply-Text, Job-UUID, and Content-Length if applicable. It then writes the
// response body to the writer.
func (r response) WriteTo(w io.Writer) (int64, error) {
	//nolint:errcheck // writing to buffer
	return writeTo(w, func(buf *bufio.Writer) {
		buf.WriteString("Content-Type: ")
		buf.WriteString(r.contentType)

		if r.text != "" {
			buf.WriteByte('\n')
			buf.WriteString("Reply-Text: ")
			buf.WriteString(r.text)
		}

		if r.jobUUID != "" {
			buf.WriteString("\nJob-UUID: ")
			buf.WriteString(r.jobUUID)
		}

		if length := len(r.body); length > 0 {
			buf.WriteString("\nContent-Length: ")
			buf.WriteString(strconv.Itoa(length))
			buf.WriteString("\n\n")
			buf.Write(r.body)
		}
	})
}

// String returns the string representation of the response.
func (r response) String() string {
	return wstr(r)
}

// LogValue returns a slog.Value object that represents the log attributes for the response.
func (r response) LogValue() slog.Value {
	attr := make([]slog.Attr, 0, 3)
	attr = append(attr, slog.String("type", r.contentType))

	if r.jobUUID != "" {
		attr = append(attr, slog.String("job-uuid", r.jobUUID))
	}

	if err := r.AsErr(); err != nil {
		attr = append(attr, slog.String("error", err.Error()))
	} else if length := r.ContentLength(); length > 0 {
		attr = append(attr, slog.Int("length", length))
	}

	return slog.GroupValue(attr...)
}

// isZero checks if the response is zero.
func (r response) isZero() bool {
	return r.contentType == ""
}

// toEvent converts a response to an Event struct.
//
// It expects the response to have a content type of "text/event-plain".
// It returns an Event struct and an error if the content type is not supported.
func (r response) toEvent() (Event, error) {
	if ct := r.ContentType(); ct != eventPlain {
		return Event{}, fmt.Errorf("unsupported event content type: %s", ct)
	}

	return parseEvent(r.body)
}
