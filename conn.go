package esl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

type conn struct {
	r   *bufio.Reader
	w   *bufio.Writer
	mu  sync.Mutex // write lock
	log *slog.Logger
}

// newConn creates a new `conn` object.
func newConn(rw io.ReadWriter, log *slog.Logger) *conn {
	if log == nil {
		log = nopLogger
	}

	return &conn{
		r:   bufio.NewReader(rw),
		w:   bufio.NewWriter(rw),
		mu:  sync.Mutex{},
		log: log,
	}
}

// Write writes a command to the connection.
func (c *conn) Write(cmd command) error {
	if cmd.IsZero() {
		return nil
	}

	c.log.Info("esl: send", slog.Any("cmd", cmd))

	c.mu.Lock()
	defer c.mu.Unlock()

	cmd.WriteTo(c.w)        //nolint:errcheck // write to buffer
	c.w.WriteString("\n\n") //nolint:errcheck // write to buffer

	if err := c.w.Flush(); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// Read reads the response from the connection.
//
// It reads the response line by line from the connection and
// parses the header values. It handles different header keys
// such as "Content-Type", "Reply-Text", "Job-UUID", and
// "Content-Length". If the "Content-Length" header is present,
// it reads the specified number of bytes as the response body.
// Finally, it logs the received response and returns it along
// with any error encountered during the process.
func (c *conn) Read() (response, error) {
	var (
		contentLength int
		resp          response
	)

	for {
		line, err := c.readLine()
		if err != nil {
			return resp, err
		}

		if len(line) == 0 {
			if resp.isZero() {
				continue // skip empty response
			}

			break // the end of response header
		}

		idx := bytes.IndexByte(line, ':')
		if idx <= 0 {
			return resp, fmt.Errorf("malformed header line: %q", line)
		}

		key, value := string(line[:idx]), trimLeft(line[idx+1:])
		switch key {
		case "Content-Type":
			resp.contentType = value
		case "Reply-Text":
			resp.text = value
		case "Job-UUID":
			resp.jobUUID = value
		case "Content-Length":
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return resp, fmt.Errorf("malformed content-length: %q", value)
			}
		default:
			c.log.Warn(
				"esl: unsupported response header",
				slog.String("key", key),
				slog.String("value", value),
			)
		}
	}

	if contentLength > 0 {
		resp.body = make([]byte, contentLength)
		if _, err := io.ReadFull(c.r, resp.body); err != nil {
			return resp, fmt.Errorf("failed to read body: %w", err)
		}
	}

	c.log.Info("esl: receive", slog.Any("response", resp))

	return resp, nil
}

// readLine reads a line from the conn's reader.
func (c *conn) readLine() ([]byte, error) {
	var fullLine []byte // to accumulate full line

	for {
		line, more, err := c.r.ReadLine()
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		if fullLine == nil && !more {
			return line, nil // the whole line is read at once
		}

		fullLine = append(fullLine, line...) // accumulate

		if !more {
			return fullLine, nil // it's the end of line
		}
	}
}

// Authentication errors.
var (
	ErrMissingAuthRequest = errors.New("missing auth request")
	ErrAccessDenied       = errors.New("access denied")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrTimeout            = errors.New("timeout")
)

// Auth authenticates the connection using the provided password.
func (c *conn) Auth(password string) error {
	resp, err := c.Read()
	if err != nil {
		return ErrMissingAuthRequest
	}

	switch contentType := resp.ContentType(); contentType {
	case "auth/request": // OK
	case "text/rude-rejection": // access denied
		return ErrAccessDenied
	case disconnectNotice: // not authorized
		return io.EOF
	default:
		return fmt.Errorf("unexpected auth request content type: %s", contentType)
	}

	if err := c.Write(cmd("auth", password)); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	resp, err = c.Read()
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if ct := resp.ContentType(); ct != commandReply {
		return fmt.Errorf("unexpected auth response content type: %s", ct)
	}

	if resp.Text() != "+OK accepted" {
		return ErrInvalidPassword
	}

	return nil
}

// AuthTimeout performs an authentication with a timeout.
func (c *conn) AuthTimeout(password string, timeout time.Duration) error {
	chErr := make(chan error, 1)
	go func() {
		chErr <- c.Auth(password)
		close(chErr)
	}()

	timer := time.NewTimer(timeout)
	select {
	case err := <-chErr:
		timer.Stop()

		return err
	case <-timer.C:
		return ErrTimeout
	}
}
