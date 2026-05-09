package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/logging"
)

func TestNew_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New(&buf, slog.LevelDebug)

	log.Debug("debug message", "key", "val")
	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message")

	out := buf.String()
	for _, want := range []string{"debug message", "info message", "warn message", "error message"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot: %s", want, out)
		}
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New(&buf, slog.LevelWarn)

	log.Debug("should be filtered")
	log.Info("should be filtered")
	log.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should be filtered") {
		t.Errorf("expected debug/info to be filtered, got: %s", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("expected warn to appear, got: %s", out)
	}
}

func TestDiscard_Silences(t *testing.T) {
	log := logging.Discard()
	// Must not panic; output is silently dropped.
	log.Debug("x")
	log.Info("x")
	log.Warn("x")
	log.Error("x")
}

func TestNewText_WritesText(t *testing.T) {
	var buf bytes.Buffer
	log := logging.NewText(&buf, slog.LevelInfo)
	log.Info("hello text", "count", 42)

	out := buf.String()
	if !strings.Contains(out, "hello text") {
		t.Errorf("text output missing message, got: %s", out)
	}
}
