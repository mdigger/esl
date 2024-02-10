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
	//nolint:errcheck // return 0 on error
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
	return writeTo(w, func(w *bufio.Writer) {
		for _, k := range keys {
			w.WriteString(k)
			w.WriteString(": ")
			// Since messaging format is similar to RFC 2822, if you are using any
			// libraries that follow the line wrapping recommendation of RFC 2822 then
			// make sure that you disable line wrapping as FreeSWITCH will ignore
			// wrapped lines.
			skipNewLines.WriteString(w, e.headers[k])
			w.WriteByte('\n')
		}
		if length := len(e.body); length > 0 {
			w.WriteString("Content-Length: ")
			w.WriteString(strconv.Itoa(length))
			w.WriteString("\n\n")
			w.Write(e.body)
		}
	})
}

// String returns a string representation of the Event.
func (e Event) String() string {
	return wstr(e)
}

// MarshalJSON is a Go function that marshals the Event to JSON.
func (e Event) MarshalJSON() ([]byte, error) {
	h := e.headers
	if len(e.body) > 0 {
		h = maps.Clone(h)
		h["_body"] = string(e.body)
	}

	return json.Marshal(h)
}

// UnmarshalJSON is a Go function that unmarshal the Event from JSON.
func (e *Event) UnmarshalJSON(data []byte) error {
	e.headers = make(map[string]string)
	if err := json.Unmarshal(data, &e.headers); err != nil {
		return err
	}

	if e.headers["_body"] != "" {
		e.body = []byte(e.headers["_body"])
		delete(e.headers, "_body")
	}

	return nil
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
var skipNewLines = strings.NewReplacer("\r\n", " ", "\n", " ")

// parseEvent parses the given byte slice as an event and returns an Event and an error.
func parseEvent(body []byte) (Event, error) {
	e := Event{
		headers: make(map[string]string, upcomingHeaderKeys(body)),
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
			return e, fmt.Errorf("malformed header line: %q", line)
		}

		key, value := string(line[:idx]), trimLeft(line[idx+1:])
		if v, err := url.PathUnescape(value); err == nil {
			value = v
		}

		e.headers[key] = value
	}

	//nolint:errcheck // use 0 as length on error
	if length, _ := strconv.Atoi(e.headers["Content-Length"]); length > 0 {
		e.body = make([]byte, length)
		if copy(e.body, body) != length {
			return e, fmt.Errorf("failed to read body: %w", io.ErrUnexpectedEOF)
		}
	}

	return e, nil
}

// upcomingHeaderKeys returns the number of upcoming header keys in the given byte slice.
func upcomingHeaderKeys(body []byte) (n int) {
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
