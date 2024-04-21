package esl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Event represents an ESL event with headers and a body.
type Event struct {
	headers map[string]string
	body    []byte
}

// NewEvent returns a new Event with the given name, headers and body.
//
// It panics if the name is empty or if the name is CUSTOM without the Event-Subclass name.
func NewEvent(name string, headers map[string]string, body []byte) Event {
	//nolint:forbidigo
	switch name {
	case "":
		panic("event name cannot be empty")
	case "CUSTOM":
		panic("event name cannot be CUSTOM without Event-Subclass name")
	}

	if name, ok := isCustomEvent(name); ok {
		headers["Event-Name"] = "CUSTOM"
		headers["Event-Subclass"] = name
	} else {
		headers["Event-Name"] = name
		delete(headers, "Event-Subclass")
	}

	if _, ok := headers["Content-Length"]; !ok {
		delete(headers, "Content-Length")
	}

	return Event{
		headers: headers,
		body:    body,
	}
}

// Get returns the value associated with the given key from the Event's headers.
func (e Event) Get(key string) string {
	return e.headers[key]
}

// Name returns the name of the event.
func (e Event) Name() string {
	if name := e.Get("Event-Subclass"); name != "" {
		return name
	}

	return e.Get("Event-Name")
}

// ContentType returns the content type of the event.
func (e Event) ContentType() string {
	return e.Get("Content-Type")
}

// ContentLength returns the length of the body in the Event.
func (e Event) ContentLength() int {
	return len(e.body)
}

// Sequence returns the event sequence as an int64.
func (e Event) Sequence() int64 {
	i, _ := strconv.ParseInt(e.Get("Event-Sequence"), 10, 64)

	return i
}

// Timestamp returns the timestamp of the event.
func (e Event) Timestamp() time.Time {
	ts := e.Get("Event-Date-Timestamp")
	if i, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.UnixMicro(i)
	}

	return time.Time{}
}

// Variable returns the value of the variable with the given name.
func (e Event) Variable(name string) string {
	return e.Get("variable_" + name)
}

// Body returns the body of the event as a string.
func (e Event) Body() string {
	return string(e.body)
}

// WriteTo writes the event to the given writer.
func (e Event) WriteTo(w io.Writer) (int64, error) {
	keys := make([]string, 0, len(e.headers))

	for k := range e.headers {
		if strings.EqualFold(k, "Content-Length") {
			continue // ignore content-length
		}

		keys = append(keys, k)
	}

	slices.Sort(keys)

	//nolint:errcheck // writing to buffer
	return writeTo(w, func(buf *bufio.Writer) {
		for _, key := range keys {
			buf.WriteString(key)
			buf.WriteString(": ")
			// Since messaging format is similar to RFC 2822, if you are using any
			// libraries that follow the line wrapping recommendation of RFC 2822 then
			// make sure that you disable line wrapping as FreeSWITCH will ignore
			// wrapped lines.
			skipNewLines.WriteString(buf, e.headers[key])
			buf.WriteByte('\n')
		}

		if length := len(e.body); length > 0 {
			buf.WriteString("Content-Length: ")
			buf.WriteString(strconv.Itoa(length))
			buf.WriteString("\n\n")
			buf.Write(e.body)
		}
	})
}

// String returns a string representation of the Event.
func (e Event) String() string {
	return writeStr(e)
}

// MarshalJSON is a Go function that marshals the Event to JSON.
func (e Event) MarshalJSON() ([]byte, error) {
	header := e.headers
	if len(e.body) > 0 {
		header = maps.Clone(header)
		header["_body"] = string(e.body)
	}

	return json.Marshal(header) //nolint:wrapcheck
}

// LogValue returns the log value of the Event.
//
// It returns a slog.Value that contains the name and sequence of the Event.
func (e Event) LogValue() slog.Value {
	attr := make([]slog.Attr, 0, 3)
	attr = append(attr,
		slog.String("name", e.Name()),
		slog.Int64("sequence", e.Sequence()),
	)

	if jobUUID := e.Get("Job-UUID"); jobUUID != "" {
		attr = append(attr, slog.String("job-uuid", jobUUID))
	}

	return slog.GroupValue(attr...)
}

// skipNewLines replaces newline characters with spaces when writing
// message headers. This ensures headers do not contain newlines, as
// required by the FreeSWITCH messaging protocol.
var skipNewLines = strings.NewReplacer("\r\n", " ", "\n", " ") //nolint:gochecknoglobals

// parseEvent parses the given byte slice as an event and returns an Event and an error.
func parseEvent(body []byte) (Event, error) {
	event := Event{
		headers: make(map[string]string, upcomingHeaderKeys(body)),
		body:    nil,
	}

	for len(body) > 0 {
		var line []byte
		if i := bytes.IndexByte(body, '\n'); i >= 0 {
			line, body = body[:i], body[i+1:]
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\r') {
			break // the end of headers
		}

		idx := bytes.IndexByte(line, ':')
		if idx <= 0 {
			return event, fmt.Errorf("malformed header line: %q", line)
		}

		key, value := string(line[:idx]), trimLeft(line[idx+1:])
		if v, err := url.PathUnescape(value); err == nil {
			value = v
		}

		event.headers[key] = value
	}

	if length, _ := strconv.Atoi(event.headers["Content-Length"]); length > 0 {
		event.body = make([]byte, length)
		if copy(event.body, body) != length {
			return event, fmt.Errorf("failed to read body: %w", io.ErrUnexpectedEOF)
		}
	}

	return event, nil
}

// upcomingHeaderKeys returns the number of upcoming header keys in the given byte slice.
func upcomingHeaderKeys(body []byte) int {
	var n int

	for len(body) > 0 && n < 1000 {
		var line []byte
		if i := bytes.IndexByte(body, '\n'); i >= 0 {
			line, body = body[:i], body[i+1:]
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\r') {
			break
		}

		n++
	}

	return n
}
