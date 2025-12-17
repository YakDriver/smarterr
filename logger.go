// logger.go
// User-facing logging infrastructure for smarterr consumers.
//
// This file defines the Logger interface and adapters for emitting logs that are intended for
// end users of smarterr (e.g., error, warning, and info messages as configured by the user).
// This is distinct from internal smarterr debugging, which is handled elsewhere.

package smarterr

import (
	"context"
	"fmt"
	"log"

	tflog "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Logger is the interface for user-facing logs emitted by smarterr.
// These logs are intended for consumers of smarterr, not for internal debugging.
type Logger interface {
	Debug(ctx context.Context, msg string, keyvals map[string]any)
	Info(ctx context.Context, msg string, keyvals map[string]any)
	Warn(ctx context.Context, msg string, keyvals map[string]any)
	Error(ctx context.Context, msg string, keyvals map[string]any)
}

// globalLogger is the user-facing logger used by smarterr for emitting logs to consumers.
var globalLogger Logger

// SetLogger sets the global user-facing logger for smarterr.
func SetLogger(logger Logger) {
	globalLogger = logger
}

// StdLogger is an adapter that emits user-facing logs using the standard Go log package.
type StdLogger struct{}

func (l StdLogger) Debug(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("DEBUG", msg, keyvals)
}
func (l StdLogger) Info(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("INFO", msg, keyvals)
}
func (l StdLogger) Warn(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("WARN", msg, keyvals)
}
func (l StdLogger) Error(ctx context.Context, msg string, keyvals map[string]any) {
	logPrint("ERROR", msg, keyvals)
}

// logPrint formats and prints a user-facing log message using the standard log package.
func logPrint(level, msg string, keyvals map[string]any) {
	logMsg := level + ": " + msg
	if len(keyvals) > 0 {
		var parts []string
		for k, v := range keyvals {
			parts = append(parts, k+"="+fmt.Sprint(v))
		}
		logMsg += " | " + strings.Join(parts, " ")
	}
	// DO NOT use this for internal smarterr debug logs! This is strictly for user-facing logs.
	log.Println(logMsg)
}

// TFLogLogger is an adapter that emits user-facing logs using Terraform's tflog package.
type TFLogLogger struct{}

func (l TFLogLogger) Debug(ctx context.Context, msg string, keyvals map[string]any) {
	tflog.Debug(ctx, msg, keyvals)
}
func (l TFLogLogger) Info(ctx context.Context, msg string, keyvals map[string]any) {
	tflog.Info(ctx, msg, keyvals)
}
func (l TFLogLogger) Warn(ctx context.Context, msg string, keyvals map[string]any) {
	tflog.Warn(ctx, msg, keyvals)
}
func (l TFLogLogger) Error(ctx context.Context, msg string, keyvals map[string]any) {
	tflog.Error(ctx, msg, keyvals)
}
