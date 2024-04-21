package esl

import (
	"context"
	"io"
	"log/slog"
)

// Option is a function type used to modify configuration options.
type Option func(*config)

// WithEvents returns an Option that sets the events channel of a config.
//
// If the autoClose parameter is specified, the channel will be automatically
// closed when the connection to the server is closed.
func WithEvents(events chan<- Event, autoClose ...bool) Option {
	return func(c *config) {
		c.events = events
		c.autoClose = len(autoClose) > 0 && autoClose[0]
	}
}

// WithLog returns an Option that sets the logger for the configuration.
func WithLog(log *slog.Logger) Option {
	return func(c *config) {
		c.log = log
	}
}

// WithDumpIn sets the writer to record all incoming messages.
// It is used for logging during debugging.
func WithDumpIn(w io.Writer) Option {
	return func(c *config) {
		c.r = w
	}
}

// WithDumpOut sets the writer to record all outgoing commands.
// It is used for logging during debugging.
func WithDumpOut(w io.Writer) Option {
	return func(c *config) {
		c.w = w
	}
}

type config struct {
	events    chan<- Event
	autoClose bool // automatically close the events channel on disconnect
	log       *slog.Logger
	r, w      io.Writer // in/out dumper
}

// getConfig returns a config object based on the provided options.
//
// It takes in a variadic parameter opts of type Option, which represents
// the configuration options for the config object.
//
// The function iterates over the options and applies each option to the
// config object. If the log field of the config object is nil, it sets it
// to the nopLogger.
//
// The function returns the generated config object.
func getConfig(opts ...Option) config {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.log == nil {
		cfg.log = nopLogger
	}

	return cfg
}

// dumper returns an io.ReadWriter that performs additional operations on the provided io.ReadWriter based on the
// configuration provided.
func (cfg config) dumper(rw io.ReadWriter) io.ReadWriter {
	if cfg.r == nil && cfg.w == nil {
		return rw
	}

	dump := struct {
		io.Reader
		io.Writer
	}{rw, rw}

	if cfg.r != nil {
		dump.Reader = io.TeeReader(rw, cfg.r)
	}

	if cfg.w != nil {
		dump.Writer = io.MultiWriter(rw, cfg.w)
	}

	return &dump
}

var nopLogger = slog.New(new(nopLogHandler)) //nolint:gochecknoglobals

// nopLogHandler is an empty struct that implements the slog.Handler interface.
// It is used to create a no-op logger that ignores all log records.
type nopLogHandler struct{}

var _ slog.Handler = (*nopLogHandler)(nil)

// Enabled returns false to indicate that logging is disabled for this handler.
// It ignores the context and level parameters.
func (*nopLogHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

// Handle implements slog.Handler. It does nothing and always returns nil.
// This allows nullLogHandler to act as a no-op logger that ignores all log records.
func (*nopLogHandler) Handle(context.Context, slog.Record) error {
	return nil
}

// WithAttrs returns the nullLogHandler unchanged.
// It ignores the provided attributes since logging is disabled.
func (h *nopLogHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

// WithGroup returns the nullLogHandler unchanged.
// It ignores the provided group name since logging is disabled.
func (h *nopLogHandler) WithGroup(string) slog.Handler {
	return h
}
