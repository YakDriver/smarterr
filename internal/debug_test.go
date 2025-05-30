package internal

import (
	"bytes"
	"testing"
)

func TestGlobalLogger(t *testing.T) {
	// Test default global logger
	logger := GetGlobalLogger()
	if _, ok := logger.(*StreamLogger); !ok {
		t.Errorf("Expected default global logger to be StreamLogger, got %T", logger)
	}

	// Test setting and getting global logger
	buf := &bytes.Buffer{}
	SetGlobalLogger(&StreamLogger{out: buf, level: "debug"})
	logger = GetGlobalLogger()
	logger.Debug("Test message")
	if buf.String() != "[DEBUG] Test message\n" {
		t.Errorf("Expected log output '[DEBUG] Test message', got %q", buf.String())
	}
}

func TestSetupLogger(t *testing.T) {
	// Test NoopLogger for silent log level
	cfg := &LoggingConfig{LogLevel: "silent"}
	logger := setupLogger(cfg)
	logger.Debug("This should not be logged")
	logger.Info("This should not be logged")
	logger.Warn("This should not be logged")
	logger.Error("This should not be logged")

	// Test StreamLogger
	buf := &bytes.Buffer{}
	cfg = &LoggingConfig{LogLevel: "debug", Output: buf}
	logger = setupLogger(cfg)
	logger.Debug("Debug message")
	if buf.String() != "[DEBUG] Debug message\n" {
		t.Errorf("Expected log output '[DEBUG] Debug message', got %q", buf.String())
	}
}

func TestFallbackMessage(t *testing.T) {
	cfg := &Config{Fallback: "detailed"}
	msg := fallbackMessage(cfg, "example_token")
	if msg != "[unresolved token: example_token]" {
		t.Errorf("Expected detailed fallback message, got %q", msg)
	}

	cfg.Fallback = "basic"
	msg = fallbackMessage(cfg, "example_token")
	if msg != "" {
		t.Errorf("Expected basic fallback message to be empty, got %q", msg)
	}
}
