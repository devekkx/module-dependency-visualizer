package logging

import (
	"context"
	"io"
	"log/slog"
)

// Logger is a minimal structured logging interface that decouples callers from slog.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// New returns a Logger backed by slog writing JSON to w at the given level.
func New(w io.Writer, level slog.Level) Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return &slogLogger{l: slog.New(h)}
}

// NewText returns a Logger backed by slog writing text to w at the given level.
func NewText(w io.Writer, level slog.Level) Logger {
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return &slogLogger{l: slog.New(h)}
}

// Discard returns a Logger that silently drops all messages.
func Discard() Logger {
	return &slogLogger{l: slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))}
}

type slogLogger struct {
	l *slog.Logger
}

func (s *slogLogger) Debug(msg string, args ...any) {
	s.l.Log(context.Background(), slog.LevelDebug, msg, args...)
}

func (s *slogLogger) Info(msg string, args ...any) {
	s.l.Log(context.Background(), slog.LevelInfo, msg, args...)
}

func (s *slogLogger) Warn(msg string, args ...any) {
	s.l.Log(context.Background(), slog.LevelWarn, msg, args...)
}

func (s *slogLogger) Error(msg string, args ...any) {
	s.l.Log(context.Background(), slog.LevelError, msg, args...)
}
