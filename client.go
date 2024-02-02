package esl

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"
	"time"
)

// spell-checker:words myevents bgapi noevents nixevent sendevent

// Client represents a client FreeSWITCH connection.
type Client struct {
	conn   *conn
	chErr  chan error
	chResp chan response
	closer io.Closer
}

// Default timeout options.
var (
	DialTimeout = time.Second * 5
	AuthTimeout = time.Second * 2
)

// Connect connects to the given address with an optional password and options.
//
// The address should include the host and port. If the port is missing, the default port 8021 will be used.
// The password is optional and can be empty.
// The options are variadic and can be used to customize the connection.
//
// Returns a new Client and an error if there was a failure in connecting.
func Connect(addr, password string, opts ...Option) (*Client, error) {
	// If the address doesn't contain a port, use the default port
	if _, _, err := net.SplitHostPort(addr); err != nil {
		var addrErr *net.AddrError
		if !errors.As(err, &addrErr) || addrErr.Err != "missing port in address" {
			return nil, fmt.Errorf("bad address: %w", err)
		}

		const defaultPort = "8021"
		addr = net.JoinHostPort(addr, defaultPort)
	}

	conn, err := net.DialTimeout("tcp", addr, DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return NewClient(conn, password, opts...)
}

// NewClient creates a new Client instance.
func NewClient(rwc io.ReadWriteCloser, password string, opts ...Option) (*Client, error) {
	cfg := getConfig(opts...)

	conn := newConn(cfg.dumper(rwc), cfg.log)

	if err := conn.AuthTimeout(password, AuthTimeout); err != nil {
		rwc.Close()
		return nil, fmt.Errorf("failed to auth: %w", err)
	}

	client := &Client{
		conn:   conn,
		chErr:  make(chan error, 1),
		chResp: make(chan response),
		closer: rwc,
	}

	go client.runReader(cfg.events, cfg.autoClose)
	runtime.Gosched()

	return client, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.sendRecv(cmd("exit")) //nolint:errcheck // ignore send error
	return c.closer.Close()
}

// API sends a command to the API and returns the response body or an error.
//
// Send a FreeSWITCH API command, blocking mode. That is, the FreeSWITCH
// instance won't accept any new commands until the api command finished execution.
func (c *Client) API(command string) (string, error) {
	resp, err := c.sendRecv(cmd("api", command))
	if err != nil {
		return "", err
	}

	return resp.Body(), nil
}

// Job sends a background command and returns the job-ID.
//
// Send a FreeSWITCH API command, non-blocking mode. This will let you execute a job
// in the background.
//
// The same API commands available as with the api command, however the server
// returns immediately and is available for processing more commands.
//
// When the command is done executing, FreeSWITCH fires an event with the result
// and you can compare that to the Job-UUID to see what the result was. In order
// to receive this event, you will need to subscribe to BACKGROUND_JOB events.
func (c *Client) Job(command string) (id string, err error) {
	resp, err := c.sendRecv(cmd("bgapi", command))
	if err != nil {
		return "", err
	}

	return resp.JobUUID(), nil
}

// JobWithID sends a background command with a specified ID.
//
// Send a FreeSWITCH API command, non-blocking mode. This will let you execute a job
// in the background, and the result will be sent as an event with an indicated UUID
// to match the reply to the command.
//
// When the command is done executing, FreeSWITCH fires an event with the result
// and you can compare that to the Job-UUID to see what the result was. In order
// to receive this event, you will need to subscribe to BACKGROUND_JOB events.
func (c *Client) JobWithID(command, id string) error {
	_, err := c.sendRecv(cmd("bgapi", command).WithJobUUID(id))
	return err
}

// Subscribe is a function that subscribes the client to events with the given names.
//
// You may specify any number events on the same line that should be separated with spaces.
//
// Subsequent calls to event won't override the previous event sets.
func (c *Client) Subscribe(names ...string) error {
	cmdNames := buildEventNamesCmd(names...)
	_, err := c.sendRecv(cmd("event", cmdNames))
	return err
}

// Unsubscribe unsubscribes the client from one or more events.
//
// Suppress the specified type of event.
// If name is empty then all events will be suppressed.
func (c *Client) Unsubscribe(names ...string) (err error) {
	cmdNames := buildEventNamesCmd(names...)
	if cmdNames == eventAll {
		_, err = c.sendRecv(cmd("noevents"))
	} else {
		_, err = c.sendRecv(cmd("nixevent", cmdNames))
	}

	return err
}

// Filter performs a filter operation on the Client.
//
// Specify event types to listen for. Note, this is not a filter out but rather
// a "filter in," that is, when a filter is applied only the filtered values are received.
// Multiple filters on a socket connection are allowed.
//
// You can filter on any of the event headers. To filter for a specific channel
// you will need to use the uuid:
//
//	filter Unique-ID d29a070f-40ff-43d8-8b9d-d369b2389dfe
//
// This method is an alternative to the myevents event type. If you need only
// the events for a specific channel then use myevents, otherwise use a combination
// of filters to narrow down the events you wish to receive on the socket.
//
// To filter multiple unique IDs, you can just add another filter for events for
// each UUID. This can be useful for example if you want to receive start/stop-talking
// events for multiple users on a particular conference.
func (c *Client) Filter(eventHeader, valueToFilter string) error {
	_, err := c.sendRecv(cmd("filter", eventHeader, valueToFilter))
	return err
}

// FilterDelete removes a filter from the Client.
//
// Specify the events which you want to revoke the filter.
// filter delete can be used when some filters are applied wrongly or when there
// is no use of the filter.
func (c *Client) FilterDelete(eventHeader, valueToFilter string) error {
	_, err := c.sendRecv(cmd("filter delete", eventHeader, valueToFilter))
	return err
}

// The 'myevents' subscription allows your inbound socket connection to behave
// like an outbound socket connect. It will "lock on" to the events for a particular
// uuid and will ignore all other events, closing the socket when the channel goes
// away or closing the channel when the socket disconnects and all applications
// have finished executing.
//
// Once the socket connection has locked on to the events for this particular
// uuid it will NEVER see any events that are not related to the channel, even
// if subsequent event commands are sent. If you need to monitor a specific
// channel/uuid and you need watch for other events as well then it is best to
// use a filter.
func (c *Client) MyEvent(uuid string) error {
	_, err := c.sendRecv(cmd("myevents", uuid))
	return err
}

// spell-checker:words inputcallback gtalk

// The divert_events switch is available to allow events that an embedded script
// would expect to get in the inputcallback to be diverted to the event socket.
//
// An inputcallback can be registered in an embedded script using setInputCallback().
// Setting divert_events to "on" can be used for chat messages like gtalk channel,
// ASR events and others.
func (c *Client) DivertEvents(on ...bool) error {
	val := "off"
	if len(on) > 0 && on[0] {
		val = "on"
	}

	_, err := c.sendRecv(cmd("divert_events", val))
	return err
}

// Send an event into the event system.
func (c *Client) SendEvent(name string, headers map[string]string, body string) error {
	_, err := c.sendRecv(
		cmd("sendevent", name).WithMessage(headers, body))
	return err
}

// SendMsg is used to control the behavior of FreeSWITCH. UUID is mandatory,
// and it refers to a specific call (i.e., a channel or call leg or session).
func (c *Client) SendMsg(uuid string, headers map[string]string, body string) error {
	_, err := c.sendRecv(
		cmd("sendmsg", uuid).WithMessage(headers, body))
	return err
}

// runReader is a method of the Client struct that reads responses from the connection and handles them accordingly.
func (c *Client) runReader(events chan<- Event, autoClose bool) {
	c.conn.log.Info("esl: run response reading")
	defer func() {
		close(c.chResp)
		close(c.chErr)
		if autoClose && events != nil {
			close(events)
		}
		c.conn.log.Info("esl: response reader stopped")
	}()

	for {
		resp, err := c.conn.Read()
		if err != nil {
			c.chErr <- err
			return // break on read error
		}

		switch ct := resp.ContentType(); ct {
		case "api/response", "command/reply":
			c.chResp <- resp

		case "text/event-plain":
			if events == nil {
				continue // ignore events if no events channel is provided
			}

			event, err := resp.toEvent()
			if err != nil {
				c.conn.log.Error("esl: failed to parse event",
					slog.String("err", err.Error()))
				continue // ignore bad event
			}

			c.conn.log.Info("esl: handle", slog.Any("event", event))
			events <- event

		case "text/disconnect-notice":
			return // disconnect

		default:
			c.conn.log.Warn("esl: unexpected response",
				slog.String("content-type", ct))
		}
	}
}

// sendRecv sends a command to the server and returns the response.
func (c *Client) sendRecv(cmd command) (response, error) {
	if err := c.conn.Write(cmd); err != nil {
		return response{}, err
	}

	return c.read()
}

// read reads the response from the client's channel and returns it along with any error.
func (c *Client) read() (response, error) {
	select {
	case err, ok := <-c.chErr:
		if ok {
			return response{}, err
		}
		return response{}, io.EOF // connection closed

	case resp := <-c.chResp:
		if err := resp.AsErr(); err != nil {
			return response{}, err // response with error message
		}
		return resp, nil
	}
}
