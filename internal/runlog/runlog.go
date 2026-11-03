// Package runlog keeps a file log of a run. The report says what a run found; the
// log says what happened, in order, with every request whatever --status keeps.
package runlog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"

	"fuzzy/internal/fuzzer"
)

// Logger writes one logfmt line per event, which grep and most log tools read.
// Each line goes straight to the file, so a killed run keeps its log up to the
// end, and values are quoted, so a response can't forge a line or reach a terminal
// as raw escapes. A nil Logger ignores everything, which is a run without --log.
type Logger struct {
	file *os.File
	out  *slog.Logger
	sink *firstError
}

// Open appends to the file at path, creating it if needed. Appending lets a log
// keep the history of several runs, each starting with a "run started" line.
func Open(path string) (*Logger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	sink := &firstError{w: file}
	return &Logger{file: file, out: slog.New(slog.NewTextHandler(sink, nil)), sink: sink}, nil
}

// Info logs an event, with its attributes as key and value pairs.
func (l *Logger) Info(message string, attributes ...any) { l.log(slog.LevelInfo, message, attributes) }

// Warn logs something that went wrong without stopping the run.
func (l *Logger) Warn(message string, attributes ...any) { l.log(slog.LevelWarn, message, attributes) }

// Error logs a failure.
func (l *Logger) Error(message string, attributes ...any) {
	l.log(slog.LevelError, message, attributes)
}

func (l *Logger) log(level slog.Level, message string, attributes []any) {
	if l != nil {
		l.out.Log(context.Background(), level, message, attributes...)
	}
}

// Result logs one call. Server errors and calls without a response are the errors
// here: a 4xx is an everyday answer to a fuzzer.
func (l *Logger) Result(r fuzzer.Result) {
	level := slog.LevelInfo
	status := strconv.Itoa(r.Status)
	switch {
	case r.Status == fuzzer.TransportError:
		level, status = slog.LevelError, "ERR"
	case r.Status >= 500:
		level = slog.LevelError
	}
	l.log(level, "request", []any{
		"endpoint", r.Endpoint, "status", status, "duration", r.Duration, "message", r.Message, "input", r.Input,
	})
}

// Close closes the file and returns the first error of the whole log, if any. It
// can be called more than once.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	if l.file != nil {
		if err := l.file.Close(); err != nil && l.sink.err == nil {
			l.sink.err = err
		}
		l.file = nil
	}
	return l.sink.err
}

// firstError remembers a failed write: slog drops them, and a full disk shouldn't
// go unnoticed until someone needs the log.
type firstError struct {
	w   io.Writer
	err error
}

func (f *firstError) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if err != nil && f.err == nil {
		f.err = err
	}
	return n, err
}
