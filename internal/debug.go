package internal

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

var (
	globalLogger Logger
	loggerMutex  sync.Mutex
)

// SetGlobalLogger sets the global logger for smarterr.
func SetGlobalLogger(logger Logger) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	globalLogger = logger
}

// GetGlobalLogger retrieves the global logger.
// The default logger is a StreamLogger that writes to os.Stderr.
// Once Config has been loaded, this function will return the configured logger.
func GetGlobalLogger() Logger {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	if globalLogger == nil {
		globalLogger = &StreamLogger{out: os.Stderr, level: "error"}
	}
	return globalLogger
}

// LoggingConfig encapsulates logging-related options.
type LoggingConfig struct {
	LogLevel string    // "silent", "error", "warn", "info", "debug"
	Output   io.Writer // os.Stderr, os.Stdout, or custom
	UseGoLog bool      // If true, use log.Logger instead of direct output
}

// Logger interface defines logging methods.
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	SetLevel(level string)
}

// NoopLogger is a no-op logger for "silent" log level.
type NoopLogger struct{}

func (n *NoopLogger) Debug(format string, args ...any) {}
func (n *NoopLogger) Info(format string, args ...any)  {}
func (n *NoopLogger) Warn(format string, args ...any)  {}
func (n *NoopLogger) Error(format string, args ...any) {}
func (n *NoopLogger) SetLevel(level string)            {}

// GoLogger wraps the standard log.Logger.
type GoLogger struct {
	log   *log.Logger
	level string
}

func (g *GoLogger) Debug(format string, args ...any) {
	if g.level == "debug" {
		g.log.Printf("[DEBUG] "+format, args...)
	}
}

func (g *GoLogger) Info(format string, args ...any) {
	if g.level == "info" || g.level == "debug" {
		g.log.Printf("[INFO] "+format, args...)
	}
}

func (g *GoLogger) Warn(format string, args ...any) {
	if g.level == "warn" || g.level == "info" || g.level == "debug" {
		g.log.Printf("[WARN] "+format, args...)
	}
}

func (g *GoLogger) Error(format string, args ...any) {
	if g.level != "silent" {
		g.log.Printf("[ERROR] "+format, args...)
	}
}

func (g *GoLogger) SetLevel(level string) {
	g.level = level
}

// StreamLogger writes logs directly to an io.Writer.
type StreamLogger struct {
	out   io.Writer
	level string
}

func (s *StreamLogger) Debug(format string, args ...any) {
	if s.level == "debug" {
		fmt.Fprintf(s.out, "[DEBUG] "+format+"\n", args...)
	}
}

func (s *StreamLogger) Info(format string, args ...any) {
	if s.level == "info" || s.level == "debug" {
		fmt.Fprintf(s.out, "[INFO] "+format+"\n", args...)
	}
}

func (s *StreamLogger) Warn(format string, args ...any) {
	if s.level == "warn" || s.level == "info" || s.level == "debug" {
		fmt.Fprintf(s.out, "[WARN] "+format+"\n", args...)
	}
}

func (s *StreamLogger) Error(format string, args ...any) {
	if s.level != "silent" {
		fmt.Fprintf(s.out, "[ERROR] "+format+"\n", args...)
	}
}

func (s *StreamLogger) SetLevel(level string) {
	s.level = level
}

// setupLogger initializes a Logger based on LoggingConfig.
func setupLogger(cfg *LoggingConfig) Logger {
	if cfg == nil || cfg.LogLevel == "silent" {
		return &NoopLogger{}
	}

	validLevels := map[string]bool{"silent": true, "error": true, "warn": true, "info": true, "debug": true}
	if !validLevels[cfg.LogLevel] {
		cfg.LogLevel = "info" // Default log level
	}

	var logger Logger
	if cfg.UseGoLog {
		goLogger := log.New(cfg.Output, "[smarterr] ", log.LstdFlags)
		logger = &GoLogger{log: goLogger, level: cfg.LogLevel}
	} else {
		logger = &StreamLogger{out: cfg.Output, level: cfg.LogLevel}
	}

	return logger
}

func logDebug(logger Logger, format string, args ...any) {
	logger.Debug(format, args...)
}

func fallbackMessage(cfg *Config, tokenName string) string {
	switch cfg.Fallback {
	case "detailed":
		return fmt.Sprintf("[unresolved token: %s]", tokenName)
	case "basic":
		fallthrough
	default:
		return ""
	}
}
